package web

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

var (
	funcMap = template.FuncMap{
		"pad": func(totalWidth int, padValues ...any) string {
			var stringIn string = ""
			var padString string = " "

			if len(padValues) > 0 {
				stringIn = fmt.Sprintf(`%v`, padValues[0])
				if len(padValues) > 1 {
					padString = fmt.Sprintf(`%v`, padValues[1])
				}
			}

			if len(stringIn) >= totalWidth {
				return stringIn
			}
			paddingLength := totalWidth - len(stringIn)
			leftPad := paddingLength >> 1
			if leftPad < 1 {
				return stringIn
			}
			if paddingLength-leftPad < 1 {
				return strings.Repeat(padString, leftPad) + stringIn
			}
			return strings.Repeat(padString, leftPad) + stringIn + strings.Repeat(padString, paddingLength-leftPad)
		},
		"lpad": func(totalWidth int, padValues ...any) string {
			var stringIn string = ""
			var padString string = " "

			if len(padValues) > 0 {
				stringIn = fmt.Sprintf(`%v`, padValues[0])
				if len(padValues) > 1 {
					padString = fmt.Sprintf(`%v`, padValues[1])
				}
			}

			if len(stringIn) >= totalWidth {
				return stringIn
			}
			paddingLength := totalWidth - len(stringIn)
			if paddingLength < 1 {
				return stringIn
			}
			return strings.Repeat(padString, paddingLength) + stringIn
		},
		"rpad": func(totalWidth int, padValues ...any) string {
			var stringIn string = ""
			var padString string = " "

			if len(padValues) > 0 {
				stringIn = fmt.Sprintf(`%v`, padValues[0])
				if len(padValues) > 1 {
					padString = fmt.Sprintf(`%v`, padValues[1])
				}
			}

			if len(stringIn) >= totalWidth {
				return stringIn
			}
			paddingLength := totalWidth - len(stringIn)
			if paddingLength < 1 {
				return stringIn
			}
			return stringIn + strings.Repeat(padString, paddingLength)
		},
		"join": func(items []string, sep string) string { return strings.Join(items, sep) },
		"activeTelnetPorts": func(ports []string) []string {
			active := make([]string, 0, len(ports))
			for _, p := range ports {
				if n, err := strconv.Atoi(p); err == nil && n > 0 {
					active = append(active, p)
				}
			}
			return active
		},
		"lte": func(a, b int) bool { return a <= b },
		"gte": func(a, b int) bool { return a >= b },
		"lt":  func(a, b int) bool { return a < b },
		//"gt":   func(a, b int) bool { return a > b },
		"uc":  func(s string) string { return strings.Title(s) },
		"lc":  func(s string) string { return strings.ToLower(s) },
		"add": func(num int, amt int) int { return num + amt },
		"sub": func(num int, amt int) int { return num - amt },
		"mul": func(num int, amt int) int { return num * amt },
		"intRange": func(start, end int) []int {
			n := end - start + 1
			result := make([]int, n)
			for i := 0; i < n; i++ {
				result[i] = start + i
			}
			return result
		},
		"escapehtml": func(str string) string {
			return html.EscapeString(str)
		},
		"lowercase": func(str string) string {
			return strings.ToLower(str)
		},
		"now": func() int64 { return time.Now().UnixMilli() },
		"fileMtime": func(relPath string) int64 {
			adminRoot := configs.GetFilePathsConfig().AdminHtml.String()
			info, err := os.Stat(filepath.Join(adminRoot, relPath))
			if err != nil {
				return 0
			}
			return info.ModTime().Unix()
		},
		"sshEnabled": func(c configs.Config) bool {
			return int(c.Network.SSHPort) > 0 && c.FilePaths.SSHHostKeyFile != ``
		},
		"getconfig": func() configs.Config {
			return configs.GetConfig()
		},
		"httpsDiagnosticHost": func(host string) string {
			return httpsDiagnosticHost(host)
		},
		"httpsUsesExampleHost": func(host string) bool {
			return httpsUsesExampleHost(host)
		},
	}
)

func publicAssetBase(r *http.Request, cdnBase string) string {
	cdnBase = strings.TrimSpace(strings.TrimRight(cdnBase, "/"))
	if cdnBase == "" {
		return ""
	}

	cdnURL, err := url.Parse(cdnBase)
	if err != nil {
		return ""
	}

	if requestUsesHTTPS(r) && strings.EqualFold(cdnURL.Scheme, "http") {
		// Do not emit insecure CDN URLs into HTTPS pages; browsers will block
		// them as mixed content, so local same-origin assets are the safe fallback.
		return ""
	}

	return cdnBase
}

func requestUsesHTTPS(r *http.Request) bool {
	if r == nil {
		return false
	}

	if r.TLS != nil {
		return true
	}

	if strings.EqualFold(r.URL.Scheme, "https") {
		return true
	}

	if proto := forwardedProto(r.Header.Get("X-Forwarded-Proto")); proto != "" {
		return strings.EqualFold(proto, "https")
	}

	if proto := forwardedHeaderProto(r.Header.Values("Forwarded")); proto != "" {
		return strings.EqualFold(proto, "https")
	}

	return false
}

func forwardedProto(value string) string {
	if value == "" {
		return ""
	}

	part, _, _ := strings.Cut(value, ",")
	return strings.TrimSpace(part)
}

func forwardedHeaderProto(values []string) string {
	for _, value := range values {
		for _, field := range strings.Split(value, ",") {
			for _, pair := range strings.Split(field, ";") {
				key, rawValue, ok := strings.Cut(pair, "=")
				if !ok || !strings.EqualFold(strings.TrimSpace(key), "proto") {
					continue
				}

				return strings.Trim(strings.TrimSpace(rawValue), `"`)
			}
		}
	}

	return ""
}
