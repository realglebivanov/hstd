package server

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/link"
	"golang.org/x/crypto/bcrypt"
)

func (s *Server) HandleAdminReq(w http.ResponseWriter, r *http.Request) {
	_, pass, ok := r.BasicAuth()
	if !ok || bcrypt.CompareHashAndPassword([]byte(s.adminPasswordHash), []byte(pass)) != nil {
		w.Header().Set("WWW-Authenticate", `Basic realm="admin"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.adminPage(w)
	case http.MethodPost:
		s.adminAction(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) adminPage(w http.ResponseWriter) {
	links, err := s.db.List(hstdlib.XrayClientCount)
	if err != nil {
		log.Printf("admin list: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	type linkRow struct {
		Index   int
		URL     string
		IPs     string
		Enabled bool
		Comment string
	}

	var rows []linkRow
	for _, l := range links {
		hex, err := link.New(l.Index, s.rootSecret).Marshal()
		if err != nil {
			log.Printf("admin marshal link %d: %v", l.Index, err)
			continue
		}
		rows = append(rows, linkRow{
			Index:   l.Index,
			URL:     fmt.Sprintf("https://x.hstd.space:8080/%s", hex),
			IPs:     l.IPs,
			Enabled: l.Enabled,
			Comment: l.Comment,
		})
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := adminTmpl.Execute(w, rows); err != nil {
		log.Printf("execute admin tpl: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

func (s *Server) adminAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	idx, err := strconv.Atoi(r.FormValue("index"))
	if err != nil {
		http.Error(w, "bad index", http.StatusBadRequest)
		return
	}

	switch r.FormValue("action") {
	case "enable":
		err = s.db.SetEnabled(idx, true)
	case "disable":
		err = s.db.SetEnabled(idx, false)
	case "comment":
		err = s.db.SetComment(idx, r.FormValue("comment"))
	default:
		http.Error(w, "unknown action", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("admin action: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/", http.StatusSeeOther)
}

var adminTmpl = template.Must(template.New("admin").Parse(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>subsrv admin</title>
<style>
body { font-family: monospace; margin: 2em; }
table { border-collapse: collapse; width: 100%; }
th, td { border: 1px solid #ccc; padding: 6px 10px; text-align: left; }
th { background: #f5f5f5; }
tr.disabled td { color: #999; }
.url { font-size: 0.85em; word-break: break-all; }
form { display: inline; }
input[type=text] { width: 150px; }
</style>
</head>
<body>
<h2>subsrv links</h2>
{{if not .}}<p>No links tracked yet.</p>{{else}}
<table>
<tr><th>Idx</th><th>URL</th><th>IPs</th><th>Enabled</th><th>Comment</th><th>Actions</th></tr>
{{range .}}
<tr{{if not .Enabled}} class="disabled"{{end}}>
<td>{{.Index}}</td>
<td class="url">{{.URL}}</td>
<td>{{.IPs}}</td>
<td>{{if .Enabled}}yes{{else}}no{{end}}</td>
<td>{{.Comment}}</td>
<td>
{{if .Enabled}}
<form method="post"><input type="hidden" name="index" value="{{.Index}}"><input type="hidden" name="action" value="disable"><button>Disable</button></form>
{{else}}
<form method="post"><input type="hidden" name="index" value="{{.Index}}"><input type="hidden" name="action" value="enable"><button>Enable</button></form>
{{end}}
<form method="post"><input type="hidden" name="index" value="{{.Index}}"><input type="hidden" name="action" value="comment"><input type="text" name="comment" value="{{.Comment}}" placeholder="comment"><button>Set</button></form>
</td>
</tr>
{{end}}
</table>
{{end}}
</body>
</html>`))
