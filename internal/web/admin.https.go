package web

import (
	"text/template"

	"net/http"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func httpsIndex(w http.ResponseWriter, r *http.Request) {
	tmlPath := configs.GetFilePathsConfig().AdminHtml.String()
	tmpl, err := template.New("https.html").Funcs(funcMap).ParseFiles(
		util.FilePath(tmlPath+"/_header.html"),
		util.FilePath(tmlPath+"/https.html"),
		util.FilePath(tmlPath+"/_footer.html"),
	)
	if err != nil {
		mudlog.Error("HTML Template", "error", err)
		http.Error(w, "Error parsing template files", http.StatusInternalServerError)
		return
	}

	tplData := map[string]any{
		"NAV":         buildAdminNav(),
		"httpsStatus": GetHTTPSStatus(),
	}

	w.Header().Set("Cache-Control", "no-store")
	if err := tmpl.Execute(w, tplData); err != nil {
		mudlog.Error("HTML Execute", "error", err)
	}
}
