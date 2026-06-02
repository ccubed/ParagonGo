package discordactivity

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
)

var (
	//go:embed files/*
	files embed.FS
)

func init() {
	m := &DiscordActivityModule{
		plug: plugins.New(`discordactivity`, `1.0`),
	}

	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}

	m.plug.Web.WebPage(`Discord Activity`, `/discord-activity`, `discord-activity.html`, false, m.buildActivityPage)

	m.plug.Web.AdminPage("Config", "discordactivity-config", "html/admin/discordactivity-config.html", true, "Modules", "Discord Activity", "Configure the Discord Application ID and view setup instructions.", "Discord embedded Activity that lets players connect to the MUD directly inside Discord voice channels.", m.getAdminConfigTemplateData)
	m.plug.Web.AdminPage("About", "discordactivity-about", "html/admin/discordactivity-about.html", true, "Modules", "Discord Activity", "Information and version details for the Discord Activity module.", "", nil)

	m.plug.Web.AdminAPIEndpoint("GET", "discordactivity-config", m.apiGetConfig)
	m.plug.Web.AdminAPIEndpoint("PATCH", "discordactivity-config", m.apiPatchConfig, "discordactivity.write")
	m.plug.Web.RegisterPermissions(plugins.ModulePermission{
		Key:         "discordactivity.write",
		Description: "Edit Discord Activity configuration",
		Category:    "Modules",
	})
}

type DiscordActivityModule struct {
	plug *plugins.Plugin
}

func (m *DiscordActivityModule) applicationId() string {
	v := m.plug.Config.Get("ApplicationId")
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// buildActivityPage reads webclient-pure.html from the configured PublicHtml
// directory and injects the three Discord-specific modifications:
//
//  1. Discord Embedded App SDK <script> tag added before </head>
//  2. onload="init()" removed from <body> (Client.init is called after SDK ready)
//  3. Discord bootstrap block injected before </body>
//
// This avoids maintaining a separate copy of the webclient HTML.
func (m *DiscordActivityModule) buildActivityPage(r *http.Request) map[string]any {
	publicHtml := configs.GetFilePathsConfig().PublicHtml.String()
	srcPath := filepath.Join(publicHtml, "webclient-pure.html")

	raw, err := os.ReadFile(srcPath)
	if err != nil {
		mudlog.Error("discordactivity", "error", fmt.Sprintf("could not read webclient-pure.html: %v", err))
		return map[string]any{
			"DiscordActivityContent": `<html><body>Discord Activity: could not load webclient-pure.html</body></html>`,
		}
	}

	src := string(raw)

	// 1. Inject the Discord SDK script tag before </head>
	sdkTag := `    <!-- Discord Embedded App SDK (CDN, no build step required) -->
    <script src="https://cdn.jsdelivr.net/npm/@discord/embedded-app-sdk@1/dist/index.min.js"></script>

</head>`
	src = strings.Replace(src, `</head>`, sdkTag, 1)

	// 2. Remove onload="init()" - Client.init() is called after discordSdk.ready() instead.
	src = strings.Replace(src, `<body onload="init()">`, `<body>`, 1)

	// 3. Inject the Discord bootstrap block and remove the init() bridge function.
	//    The bridge block in webclient-pure.html defines function init() { Client.init(); }.
	//    We replace that specific line so the rest of the bridge functions are preserved.
	src = strings.Replace(src,
		`        function init()                  { Client.init(); }`,
		`        // init() is not defined here - Client.init() is called after discordSdk.ready() below.`,
		1,
	)

	appId := m.applicationId()
	bootstrapBlock := buildBootstrapBlock(appId)
	src = strings.Replace(src, `</body>`, bootstrapBlock+"\n</body>", 1)

	return map[string]any{
		"DiscordActivityContent": src,
	}
}

// buildBootstrapBlock returns the Discord Activity bootstrap <script> block.
// appId is injected directly as a Go string literal, not as a template variable,
// because the HTML is returned as a pre-rendered string rather than a template.
func buildBootstrapBlock(appId string) string {
	// JSON-encode the appId so any special characters are safely escaped.
	appIdJSON, _ := json.Marshal(appId)

	return `    <script>
        // -----------------------------------------------------------------------
        // Discord Activity bootstrap
        //
        // The Discord Embedded App SDK must complete its ready() handshake before
        // we initialise the MUD client. patchUrlMappings() rewrites all fetch()
        // and WebSocket() calls so they route through Discord's proxy, which means
        // the existing /ws endpoint works without any server-side changes.
        //
        // Full-screen mode (layout_mode === 1) shows the virtual window panels.
        // Compact mode (layout_mode === 2) hides them so the terminal fills the
        // entire Activity frame.
        // -----------------------------------------------------------------------
        (function () {
            'use strict';

            var applicationId = ` + string(appIdJSON) + `;

            if (!applicationId) {
                // ApplicationId not configured - show a clear error instead of a
                // broken SDK init so operators know to visit the admin config page.
                var err = document.createElement('div');
                err.style.cssText = [
                    'position:fixed', 'inset:0', 'display:flex',
                    'align-items:center', 'justify-content:center',
                    'background:#111', 'color:#f88', 'font-family:monospace',
                    'font-size:16px', 'text-align:center', 'padding:20px', 'z-index:9999',
                ].join(';');
                err.textContent = 'Discord Activity is not configured. ' +
                    'Set Modules.discordactivity.ApplicationId in the GoMud admin config.';
                document.body.appendChild(err);
                return;
            }

            var discordSdk = new DiscordSDK(applicationId);

            function applyLayoutMode(fullScreen) {
                var dockLeft  = document.getElementById('dock-left');
                var dockRight = document.getElementById('dock-right');
                if (dockLeft)  { dockLeft.style.display  = fullScreen ? '' : 'none'; }
                if (dockRight) { dockRight.style.display = fullScreen ? '' : 'none'; }
                document.querySelectorAll('.dock-slot-resize').forEach(function (el) {
                    el.style.display = fullScreen ? '' : 'none';
                });
                // Let xterm.js measure its new dimensions after the layout change.
                window.dispatchEvent(new Event('resize'));
            }

            discordSdk.ready().then(function () {
                // Rewrite fetch() and WebSocket() URLs to route through Discord's
                // proxy. The prefix '/' maps the entire origin, which covers /ws.
                discordSdk.patchUrlMappings([{ prefix: '/', target: location.host }]);

                // Subscribe to full-screen toggle events.
                discordSdk.subscribe('ACTIVITY_LAYOUT_MODE_UPDATE', function (data) {
                    // layout_mode 1 = full-screen, 2 = default embedded
                    applyLayoutMode(data.layout_mode === 1);
                });

                // Default to compact (panels hidden) until Discord fires the
                // layout mode event or the user expands to full-screen.
                applyLayoutMode(false);

                // Now that the SDK handshake is done, start the MUD client.
                Client.init();
            }).catch(function (err) {
                console.error('Discord SDK ready() failed:', err);
            });
        }());
    </script>`
}

func (m *DiscordActivityModule) getAdminConfigTemplateData(r *http.Request) map[string]any {
	netCfg := configs.GetNetworkConfig()
	scheme := "http"
	port := int(netCfg.HttpPort)
	hasHTTPS := int(netCfg.HttpsPort) > 0
	if hasHTTPS {
		scheme = "https"
		port = int(netCfg.HttpsPort)
	}
	host := r.Host
	if host == "" {
		host = fmt.Sprintf("your-server-domain:%d", port)
	}
	// Strip any port already in r.Host and use the configured port.
	for i := len(host) - 1; i >= 0; i-- {
		if host[i] == ':' {
			host = host[:i]
			break
		}
	}
	activityHost := fmt.Sprintf("%s:%d", host, port)
	activityURL := fmt.Sprintf("%s://%s/discord-activity", scheme, activityHost)
	return map[string]any{
		"ApplicationId": m.applicationId(),
		"ActivityURL":   activityURL,
		"ActivityHost":  activityHost,
		"HasHTTPS":      hasHTTPS,
	}
}

func (m *DiscordActivityModule) apiGetConfig(r *http.Request) (int, bool, any) {
	return http.StatusOK, true, map[string]any{
		"ApplicationId": m.applicationId(),
	}
}

func (m *DiscordActivityModule) apiPatchConfig(r *http.Request) (int, bool, any) {
	var body struct {
		ApplicationId string `json:"ApplicationId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return http.StatusBadRequest, false, "malformed request body: " + err.Error()
	}

	m.plug.Config.Set("ApplicationId", body.ApplicationId)

	return http.StatusOK, true, map[string]any{
		"ApplicationId": body.ApplicationId,
	}
}
