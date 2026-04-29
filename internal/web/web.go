package web

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/acme/autocert"
)

var (
	httpServer         *http.Server
	httpsServer        *http.Server
	httpsRedirectReady atomic.Bool

	// internalMux is the single ServeMux used by both the live HTTP servers and
	// the in-process InternalRequest dispatcher. All routes must be registered
	// on this mux rather than http.DefaultServeMux.
	internalMux = http.NewServeMux()

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	httpRoot = ``

	// Used to interface with plugins and request web stuff
	webPlugins WebPlugin = nil

	// defaultRegistrar is the singleton that implements ModuleAdminRegistrar.
	defaultRegistrar = &moduleAdminRegistrarImpl{}
)

// WebNav is used for the public-facing navigation.
type WebNav struct {
	Name   string
	Target string
}

// WebNavItem represents a top-level admin nav entry, optionally with sub-items
// or nested sub-menus (groups).
type WebNavItem struct {
	Name     string
	Target   string // primary href; empty if dropdown-only
	SubItems []WebNavSub
	SubMenus []WebNavItem // nested groups, each with their own SubItems
}

// WebNavSub is a single item inside a dropdown.
type WebNavSub struct {
	Label  string
	Target string
}

// ModuleAdminRegistrar is implemented by internal/web and provided to plugins
// via plugins.SetAdminRegistrar. This breaks the import cycle.
type ModuleAdminRegistrar interface {
	// RegisterAdminPage registers a module admin page.
	// htmlContent is the raw HTML read from the plugin's embedded FS.
	// navGroup, if non-empty, places the page's nav entry inside a group dropdown.
	// navParent, if non-empty, nests the page as a sub-item under that parent within the group.
	RegisterAdminPage(name, slug, htmlContent string, addToNav bool, navGroup, navParent string, dataFunc func(*http.Request) map[string]any)
	// RegisterAdminAPIEndpoint registers a module API handler.
	// handler receives the request and returns (statusCode, success, data).
	RegisterAdminAPIEndpoint(method, slug string, handler func(*http.Request) (int, bool, any))
}

// moduleAdminRegistrarImpl holds module-contributed nav and API routes.
type moduleAdminRegistrarImpl struct {
	navItems []WebNavItem
}

// GetAdminRegistrar returns the ModuleAdminRegistrar that main.go passes to
// plugins.SetAdminRegistrar.
func GetAdminRegistrar() ModuleAdminRegistrar {
	return defaultRegistrar
}

// RegisterAdminPage registers a module admin page on internalMux and records
// its nav contribution.
func (reg *moduleAdminRegistrarImpl) RegisterAdminPage(
	name, slug, htmlContent string,
	addToNav bool,
	navGroup, navParent string,
	dataFunc func(*http.Request) map[string]any,
) {
	path := "/admin/" + slug

	handler := func(w http.ResponseWriter, r *http.Request) {
		adminHtml := configs.GetFilePathsConfig().AdminHtml.String()

		tmpl, err := template.New(slug+".html").Funcs(funcMap).ParseFiles(
			adminHtml+"/_header.html",
			adminHtml+"/_footer.html",
		)
		if err != nil {
			mudlog.Error("HTML ERROR", "error", err)
			http.Error(w, "Error parsing template files", http.StatusInternalServerError)
			return
		}
		tmpl, err = tmpl.Parse(htmlContent)
		if err != nil {
			mudlog.Error("HTML ERROR", "error", err)
			http.Error(w, "Error parsing module html", http.StatusInternalServerError)
			return
		}

		templateData := map[string]any{
			"CONFIG": configs.GetConfig(),
			"STATS":  GetStats(),
			"NAV":    buildAdminNav(),
		}
		if dataFunc != nil {
			for k, v := range dataFunc(r) {
				if _, exists := templateData[k]; !exists {
					templateData[k] = v
				}
			}
		}

		w.Header().Set("Cache-Control", "no-store")
		if err := tmpl.Execute(w, templateData); err != nil {
			mudlog.Error("HTML ERROR", "action", "Execute", "error", err)
		}
	}

	internalMux.HandleFunc("GET "+path, RunWithMUDLocked(doBasicAuth(handler)))

	if !addToNav {
		return
	}

	if navGroup == "" {
		// No group: place directly in the top-level nav.
		if navParent == "" {
			// Top-level nav item with a single sub-item pointing to itself.
			reg.navItems = append(reg.navItems, WebNavItem{
				Name:   name,
				Target: path,
				SubItems: []WebNavSub{
					{Label: "View", Target: path},
				},
			})
			return
		}
		// Attach as sub-item to an existing top-level nav entry.
		for i, item := range reg.navItems {
			if item.Name == navParent {
				reg.navItems[i].SubItems = append(reg.navItems[i].SubItems, WebNavSub{
					Label:  name,
					Target: path,
				})
				return
			}
		}
		// Parent not found yet - add as top-level with the sub-item.
		reg.navItems = append(reg.navItems, WebNavItem{
			Name:   navParent,
			Target: "",
			SubItems: []WebNavSub{
				{Label: name, Target: path},
			},
		})
		return
	}

	// navGroup is set: find or create the group, then find or create the parent
	// sub-menu within it, then append the sub-item.
	groupIdx := -1
	for i, item := range reg.navItems {
		if item.Name == navGroup {
			groupIdx = i
			break
		}
	}
	if groupIdx == -1 {
		reg.navItems = append(reg.navItems, WebNavItem{Name: navGroup})
		groupIdx = len(reg.navItems) - 1
	}

	if navParent == "" {
		// No parent within the group: add a sub-menu entry for this page directly.
		reg.navItems[groupIdx].SubMenus = append(reg.navItems[groupIdx].SubMenus, WebNavItem{
			Name:   name,
			Target: path,
			SubItems: []WebNavSub{
				{Label: "View", Target: path},
			},
		})
		return
	}

	// navParent is set within the group: find or create the sub-menu for navParent.
	for i, sm := range reg.navItems[groupIdx].SubMenus {
		if sm.Name == navParent {
			reg.navItems[groupIdx].SubMenus[i].SubItems = append(
				reg.navItems[groupIdx].SubMenus[i].SubItems,
				WebNavSub{Label: name, Target: path},
			)
			return
		}
	}
	// Sub-menu for navParent not found yet - create it.
	reg.navItems[groupIdx].SubMenus = append(reg.navItems[groupIdx].SubMenus, WebNavItem{
		Name: navParent,
		SubItems: []WebNavSub{
			{Label: name, Target: path},
		},
	})
}

// RegisterAdminAPIEndpoint registers a module API endpoint on internalMux.
func (reg *moduleAdminRegistrarImpl) RegisterAdminAPIEndpoint(
	method, slug string,
	handler func(*http.Request) (int, bool, any),
) {
	path := "/admin/api/v1/" + slug

	h := func(w http.ResponseWriter, r *http.Request) {
		status, success, data := handler(r)
		writeJSON(w, status, APIResponse[any]{Success: success, Data: data})
	}

	internalMux.HandleFunc(method+" "+path, RunWithMUDLocked(doBasicAuth(h)))
}

type WebPlugin interface {
	NavLinks() map[string]string                                                    // Name=>Path pairs
	WebRequest(r *http.Request) (html string, templateData map[string]any, ok bool) // Get the first handler of a given request
}

func SetWebPlugin(wp WebPlugin) {
	webPlugins = wp
}

// serveTemplate searches for the requested file in the HTTP_ROOT,
// parses it as a template, and serves it.
func serveTemplate(w http.ResponseWriter, r *http.Request) {

	if httpRoot == "" {
		httpRoot = filepath.Clean(configs.GetFilePathsConfig().PublicHtml.String())
	}

	// Clean the path to prevent directory traversal.
	reqPath := filepath.Clean(r.URL.Path) // Example: / or /info/faq

	// Build the full file path.
	fullPath := filepath.Join(httpRoot, reqPath)

	if filepath.Base(fullPath) == "_all.js" {
		if serveConcatenatedJS(w, r, filepath.Dir(fullPath)) {
			return
		}
	}

	// If the path is a directory, look for an index.html.
	info, err := os.Stat(fullPath)
	if err != nil {
		if filepath.Ext(fullPath) != ".html" {
			fullPath += ".html"
		}
	} else if info.IsDir() {
		fullPath = filepath.Join(fullPath, "index.html")
	}

	fileExt := filepath.Ext(fullPath)
	fileBase := filepath.Base(fullPath)

	// All template files to load from the filesystem
	templateFiles := []string{}

	var pageFound bool = true

	var pluginHtml string = ``
	var pluginTplData map[string]any = nil
	var ok bool = false
	var fSize int64 = 0
	var source string = `PublicHtml folder`

	// Check if the file exists, else 404
	fInfo, err := os.Stat(fullPath)
	if err != nil {
		pageFound = false
	}

	// Allow plugin to override request
	if webPlugins != nil {
		pluginHtml, pluginTplData, ok = webPlugins.WebRequest(r)
		fSize = int64(len([]byte(pluginHtml)))
		if ok {
			source = `module`
			pageFound = true
		}
	}

	if !pageFound || len(fileBase) > 0 && fileBase[0] == '_' {
		mudlog.Info("Web", "ip", r.RemoteAddr, "ref", r.Header.Get("Referer"), "file path", fullPath, "file extension", fileExt, "error", "Not found")

		fullPath = filepath.Join(httpRoot, `404.html`)
		fInfo, err = os.Stat(fullPath)

		if err != nil {
			http.NotFound(w, r)
			return
		}

		fSize = fInfo.Size()

		w.WriteHeader(http.StatusNotFound)
	}

	// Log the request
	mudlog.Info("Web", "ip", r.RemoteAddr, "ref", r.Header.Get("Referer"), "file path", fullPath, "file extension", fileExt, "file source", source, "size", fmt.Sprintf(`%.2fk`, float64(fSize)/1024))

	// For non-HTML files, serve them statically.
	if fileExt != ".html" {
		http.ServeFile(w, r, fullPath)
		return
	}

	templateData := map[string]any{
		"REQUEST":        r,
		"PATH":           reqPath,
		"CONFIG":         configs.GetConfig(),
		"STATS":          GetStats(),
		"ASSET_BASE_URL": publicAssetBase(r, configs.GetFilePathsConfig().WebCDNLocation.String()),
		"NAV": []WebNav{
			{`Home`, `/`},
			{`Who's Online`, `/online`},
			{`Web Client`, `/webclient`},
			{`See Configuration`, `/viewconfig`},
		},
	}

	// Copy any plugin navigation
	if webPlugins != nil {

		currentNav := templateData[`NAV`].([]WebNav)

		for name, path := range webPlugins.NavLinks() {

			found := false
			for i := len(currentNav) - 1; i >= 0; i-- {

				if currentNav[i].Name == name {
					found = true
					if path == `` {
						currentNav = append(currentNav[:i], currentNav[i+1:]...)
					} else {
						currentNav[i].Target = path
					}
					break
				}

			}

			if !found {
				currentNav = append(currentNav, WebNav{name, path})
			}
		}

		templateData[`NAV`] = currentNav
	}

	// Copy over any plugin data loaded.
	for name, value := range pluginTplData {
		// Don't allow overwriting defaults
		if _, ok := templateData[name]; !ok {
			templateData[name] = value
		}
	}

	// Parse special files intended to be used as template includes
	globFiles, err := filepath.Glob(filepath.Join(httpRoot, "_*.html"))
	if err == nil {
		templateFiles = append(templateFiles, globFiles...)
	}

	// Parse special files intended to be used as template includes (from the request folder)
	requestDir := filepath.Dir(fullPath)
	if httpRoot != requestDir {
		globFiles, err = filepath.Glob(filepath.Join(requestDir, "_*.html"))
		if err == nil {
			templateFiles = append(templateFiles, globFiles...)
		}
	}

	// Add the final (actual) file

	// Parse
	tmpl := template.New(filepath.Base(fullPath)).Funcs(funcMap)

	if pluginHtml == `` {
		templateFiles = append(templateFiles, fullPath)

	}

	tmpl, err = tmpl.ParseFiles(templateFiles...)
	if err != nil {
		mudlog.Error("HTML ERROR", "action", "ParseFiles", "error", err)
		http.Error(w, "Error parsing template files", http.StatusInternalServerError)
	}

	if pluginHtml != `` {
		tmpl, err = tmpl.Parse(pluginHtml)
		if err != nil {
			mudlog.Error("HTML ERROR", "action", "Parse", "error", err)
			http.Error(w, "Error parsing plugin html", http.StatusInternalServerError)
		}
	}

	// Execute the template and write it to the response.
	w.Header().Set("Cache-Control", "no-store")
	if err := tmpl.Execute(w, templateData); err != nil {
		mudlog.Error("HTML ERROR", "action", "Execute", "error", err)
		http.Error(w, "Error executing template", http.StatusInternalServerError)
	}
}

func Listen(wg *sync.WaitGroup, webSocketHandler func(*websocket.Conn)) {
	httpsRedirectReady.Store(false)

	networkConfig := configs.GetNetworkConfig()
	filePaths := configs.GetFilePathsConfig()
	httpsPlan := resolveHTTPSPlan(networkConfig, filePaths)

	if networkConfig.HttpPort == 0 && networkConfig.HttpsPort == 0 {
		slog.Error(`Web`, "error", "No ports defined. No web server will be started.")
		return
	}

	// Routing
	// Basic homepage

	internalMux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = `/static/images/favicon.ico`
		serveTemplate(w, r)
	})

	internalMux.HandleFunc("/", serveTemplate)

	// websocket upgrade
	internalMux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("WebSocket upgrade failed:", err)
			return
		}
		defer conn.Close()

		webSocketHandler(conn)
	})

	registerAdminRoutes(internalMux)

	//
	// Https server start up
	//

	switch httpsPlan.mode {
	case httpsModeManual:
		status := newHTTPSStatus(httpsPlan, networkConfig)
		SetHTTPSStatus(status)
		logHTTPSStatus(status)

		mudlog.Info("HTTPS", "stage", "Validating public/private key pair", "Public Cert", httpsPlan.certFile, "Private Key", httpsPlan.keyFile)

		cert, err := tls.LoadX509KeyPair(httpsPlan.certFile, httpsPlan.keyFile)
		if err != nil {
			UpdateHTTPSStatus(func(status *HTTPSStatus) {
				markHTTPSStartupFailure(status, err)
			})
			mudlog.Error("HTTPS", "error", fmt.Errorf("Error loading certificate and key: %w", err))
		} else {
			if leaf, leafErr := firstLeafCertificate(&cert); leafErr == nil {
				UpdateHTTPSStatus(func(status *HTTPSStatus) {
					setCertificateInfo(status, leaf)
				})
			}

			tlsConfig := &tls.Config{
				Certificates: []tls.Certificate{cert},
			}

			httpsServer = &http.Server{
				Addr:      fmt.Sprintf(`:%d`, networkConfig.HttpsPort),
				TLSConfig: tlsConfig,
				Handler:   internalMux,
			}

			mudlog.Info("HTTPS", "stage", "Starting https server", "mode", "manual", "port", networkConfig.HttpsPort)
			startHTTPSServer(wg, httpsServer, func() {
				httpsRedirectReady.Store(true)
				UpdateHTTPSStatus(func(status *HTTPSStatus) {
					markHTTPSListenerReady(status, bool(networkConfig.HttpsRedirect))
				})
			}, func(err error) {
				httpsRedirectReady.Store(false)
				UpdateHTTPSStatus(func(status *HTTPSStatus) {
					markHTTPSStartupFailure(status, err)
					status.NextSteps = append(status.NextSteps, describeListenError(int(networkConfig.HttpsPort), err)...)
				})
				mudlog.Error("HTTPS", "error", fmt.Errorf("Error starting HTTPS web server: %w", err))
			})
		}
	case httpsModeAuto:
		status := newHTTPSStatus(httpsPlan, networkConfig)
		runAutoHTTPSPreflight(&status)
		SetHTTPSStatus(status)
		logHTTPSStatus(status)

		if err := os.MkdirAll(httpsPlan.cacheDir, 0700); err != nil {
			mudlog.Error("HTTPS", "error", fmt.Errorf("Error creating HTTPS cache dir: %w", err), "cacheDir", httpsPlan.cacheDir)
			httpsPlan.mode = httpsModeHTTPOnly
			if httpsPlan.fallbackReason == "" {
				httpsPlan.fallbackReason = fmt.Sprintf("automatic HTTPS cache directory %q is not writable", httpsPlan.cacheDir)
			}
			SetHTTPSStatus(newHTTPSStatus(httpsPlan, networkConfig))
			break
		}

		manager := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			Cache:      autocert.DirCache(httpsPlan.cacheDir),
			HostPolicy: autocert.HostWhitelist(httpsPlan.host),
			Email:      httpsPlan.email,
		}

		httpServer = &http.Server{
			Addr:    fmt.Sprintf(`:%d`, networkConfig.HttpPort),
			Handler: buildAutoHTTPHandler(manager, networkConfig, internalMux, &httpsRedirectReady),
		}

		baseTLSConfig := manager.TLSConfig()
		baseGetCertificate := baseTLSConfig.GetCertificate
		baseTLSConfig.GetCertificate = func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			if err := validateAutoHTTPSServerName(httpsPlan.host, hello.ServerName); err != nil {
				return nil, err
			}

			cert, err := baseGetCertificate(hello)
			if err == nil {
				if leaf, leafErr := firstLeafCertificate(cert); leafErr == nil {
					UpdateHTTPSStatus(func(status *HTTPSStatus) {
						setCertificateInfo(status, leaf)
						status.LastError = ""
					})
				}
			} else {
				UpdateHTTPSStatus(func(status *HTTPSStatus) {
					status.LastError = err.Error()
				})
			}
			return cert, err
		}

		httpsServer = &http.Server{
			Addr:      fmt.Sprintf(`:%d`, networkConfig.HttpsPort),
			TLSConfig: baseTLSConfig,
			Handler:   internalMux,
			ErrorLog:  log.New(automaticHTTPSLogFilter{}, "", 0),
		}

		mudlog.Info("HTTPS", "stage", "Starting https server", "mode", "automatic", "host", httpsPlan.host, "port", networkConfig.HttpsPort, "cacheDir", httpsPlan.cacheDir)
		if httpsPlan.emailNoticeNeeded {
			mudlog.Warn("HTTPS", "warning", "Automatic HTTPS is enabled without HttpsEmail; certificate expiry notices will not be sent by email.")
		}

		startHTTPSServer(wg, httpsServer, func() {
			httpsRedirectReady.Store(true)
			UpdateHTTPSStatus(func(status *HTTPSStatus) {
				markHTTPSListenerReady(status, bool(networkConfig.HttpsRedirect))
			})
		}, func(err error) {
			httpsRedirectReady.Store(false)
			UpdateHTTPSStatus(func(status *HTTPSStatus) {
				markHTTPSStartupFailure(status, err)
				status.NextSteps = append(status.NextSteps, describeListenError(int(networkConfig.HttpsPort), err)...)
			})
			mudlog.Error("HTTPS", "error", fmt.Errorf("Error starting HTTPS web server: %w", err))
		})
	case httpsModeHTTPOnly:
		status := newHTTPSStatus(httpsPlan, networkConfig)
		SetHTTPSStatus(status)
		logHTTPSStatus(status)
	}

	//
	// Http server start up
	//

	if networkConfig.HttpPort > 0 {

		if httpServer == nil {
			httpServer = &http.Server{
				Addr:    fmt.Sprintf(`:%d`, networkConfig.HttpPort),
				Handler: internalMux,
			}
		} else if httpServer.Handler == nil {
			httpServer.Handler = internalMux
		}

		if networkConfig.HttpsRedirect && httpsPlan.mode != httpsModeAuto {

			if httpsServer == nil {

				mudlog.Error("HTTP", "error", "Cannot enable https redirect. There is no https server configured/running.")

			} else {

				var redirectHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {

					target := buildHTTPSRedirectTarget(r.Host, int(networkConfig.HttpsPort), r.RequestURI)

					http.Redirect(w, r, target, http.StatusMovedPermanently)
				}

				httpServer.Handler = buildConditionalHTTPSRedirectHandler(redirectHandler, int(networkConfig.HttpsPort), internalMux, &httpsRedirectReady)

			}

		}

		// HTTP Server
		wg.Add(1)

		mudlog.Info("HTTP", "stage", "Starting http server", "port", networkConfig.HttpPort)
		go func() {
			defer wg.Done()

			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				UpdateHTTPSStatus(func(status *HTTPSStatus) {
					if status.Mode == string(httpsModeAuto) {
						markAutoHTTPSHTTPFailure(status, err)
					} else {
						status.LastError = err.Error()
					}
					status.NextSteps = append(status.NextSteps, describeListenError(int(networkConfig.HttpPort), err)...)
				})
				mudlog.Error("HTTP", "error", fmt.Errorf("Error starting web server: %w", err))
			}
		}()
	}

}

func startHTTPSServer(wg *sync.WaitGroup, server *http.Server, onBound func(), onError func(error)) {
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		onError(err)
		return
	}

	if onBound != nil {
		onBound()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		tlsListener := tls.NewListener(listener, server.TLSConfig)
		if err := server.Serve(tlsListener); err != nil && err != http.ErrServerClosed {
			onError(err)
		}
	}()
}

func buildConditionalHTTPSRedirectHandler(redirect http.Handler, httpsPort int, fallback http.Handler, redirectReady *atomic.Bool) http.Handler {
	if fallback == nil {
		fallback = internalMux
	}

	if redirect == nil {
		redirect = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			target := buildHTTPSRedirectTarget(r.Host, httpsPort, r.RequestURI)
			http.Redirect(w, r, target, http.StatusMovedPermanently)
		})
	}

	if redirectReady == nil {
		return redirect
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !redirectReady.Load() {
			fallback.ServeHTTP(w, r)
			return
		}

		redirect.ServeHTTP(w, r)
	})
}

func buildAutoHTTPHandler(manager *autocert.Manager, networkConfig configs.Network, fallback http.Handler, redirectReady *atomic.Bool) http.Handler {
	if bool(networkConfig.HttpsRedirect) {
		fallback = buildConditionalHTTPSRedirectHandler(nil, int(networkConfig.HttpsPort), fallback, redirectReady)
	}

	return manager.HTTPHandler(fallback)
}

// RunWithMUDLocked wraps a handler with the game mutex. Internal requests
// (dispatched via InternalRequest) skip locking because the caller is
// responsible for holding the lock when required.
func RunWithMUDLocked(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if IsInternalRequest(r) {
			next.ServeHTTP(w, r)
			return
		}

		util.LockMud()
		defer util.UnlockMud()

		next.ServeHTTP(w, r)
	})
}

func Shutdown() {
	httpsRedirectReady.Store(false)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if httpServer != nil {
		if err := httpServer.Shutdown(ctx); err != nil {
			mudlog.Error("HTTP", "error", fmt.Errorf("HTTP server shutdown failed: %w", err))
		} else {
			mudlog.Info("HTTP", "stage", "stopped")
		}
	}

	if httpsServer != nil {
		if err := httpsServer.Shutdown(ctx); err != nil {
			mudlog.Error("HTTPS", "error", fmt.Errorf("HTTPS server shutdown failed: %w", err))
		} else {
			mudlog.Info("HTTPS", "stage", "stopped")
		}
	}
}

func serveConcatenatedJS(w http.ResponseWriter, r *http.Request, dir string) bool {
	var names []string

	if raw := r.URL.RawQuery; raw != "" {
		seen := map[string]bool{}
		for _, name := range strings.Split(raw, ",") {
			if strings.ContainsAny(name, `/\`) || name != filepath.Base(name) {
				continue
			}
			if name == "" || name == "." || name == "_all.js" || !strings.HasSuffix(name, ".js") {
				continue
			}
			if seen[name] {
				continue
			}
			if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
				continue
			}
			seen[name] = true
			names = append(names, name)
		}
	} else {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return false
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".js") && e.Name() != "_all.js" {
				names = append(names, e.Name())
			}
		}
		sort.Strings(names)
	}

	if len(names) == 0 {
		return false
	}

	var buf bytes.Buffer
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		buf.Write(data)
		buf.WriteByte('\n')
	}

	w.Header().Set("Content-Type", "application/javascript")
	w.Write(buf.Bytes())
	return true
}

// serveAdminStaticFile serves static assets from the admin HTML directory.
// The full URL path relative to /admin/ is preserved so subdirectories work.
func serveAdminStaticFile(w http.ResponseWriter, r *http.Request) {
	adminRoot := filepath.Clean(configs.GetFilePathsConfig().AdminHtml.String())
	rel := strings.TrimPrefix(r.URL.Path, "/admin")
	fullPath := filepath.Join(adminRoot, filepath.Clean(rel))
	if filepath.Base(fullPath) == "_all.js" {
		if serveConcatenatedJS(w, r, filepath.Dir(fullPath)) {
			return
		}
	}
	http.ServeFile(w, r, fullPath)
}

func sendError(w http.ResponseWriter, r *http.Request, status int) {
	w.WriteHeader(status)
	if status == http.StatusNotFound {
		fmt.Fprint(w, "custom 404")
	}
}
