package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/server/admin/view"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/server/wsconn"
)

type wsLinksMsg struct {
	Type string      `json:"type"`
	Rows []*view.Row `json:"rows"`
}

type wsLinkUpdatedMsg struct {
	Type string    `json:"type"`
	Row  *view.Row `json:"row"`
}

type wsErrorMsg struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type wsActionReq struct {
	Type    string  `json:"type"`
	Index   int     `json:"index"`
	Comment *string `json:"comment"`
	Enabled *bool   `json:"enabled"`
}

func (s *Server) handleAdminWS(w http.ResponseWriter, r *http.Request) {
	if !s.basicAuth(w, r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	wsc, err := wsconn.Upgrade(w, r)
	if err != nil {
		slog.Warn("ws upgrade", "err", err)
		return
	}
	defer wsc.Close()
	s.broadcast.Add(wsc)
	defer s.broadcast.Remove(wsc)

	if err := s.sendLinks(wsc); err != nil {
		slog.Error("send links", "err", err)
		return
	}

	wsc.StartKeepAlive()

	for {
		msg, err := wsc.ReadMessage()
		if err != nil {
			slog.Warn("read ws message", "err", err)
			break
		}
		s.handleWSAction(wsc, msg)
	}
}

func (s *Server) handleWSAction(sender *wsconn.WSCconn, msg []byte) {
	var req wsActionReq
	if err := json.Unmarshal(msg, &req); err != nil {
		slog.Info("json", "msg", msg)
		sender.WriteJSON(wsErrorMsg{Type: "error", Message: "invalid json"})
		return
	}
	if req.Type != "update_link" {
		sender.WriteJSON(wsErrorMsg{Type: "error", Message: "unknown type"})
		return
	}

	l, err := s.db.UpdateLink(req.Index, req.Comment, req.Enabled)
	if err != nil {
		slog.Error("ws update link", "err", err)
		sender.WriteJSON(wsErrorMsg{Type: "error", Message: "internal error"})
		return
	}

	row, err := s.view.BuildRow(l)
	if err != nil {
		slog.Error("ws build row", "err", err)
		sender.WriteJSON(wsErrorMsg{Type: "error", Message: "internal error"})
		return
	}

	sender.WriteJSON(wsLinkUpdatedMsg{Type: "link_updated", Row: row})
	s.broadcast.Broadcast(row, sender)
}

func (s *Server) sendLinks(wsc *wsconn.WSCconn) error {
	links, err := s.db.List(hstdlib.XrayClientCount)
	if err != nil {
		slog.Error("ws fetch links", "err", err)
		return err
	}
	rows, err := s.view.BuildRows(links)
	if err != nil {
		slog.Error("ws build rows", "err", err)
		return err
	}
	if err := wsc.WriteJSON(wsLinksMsg{Type: "links", Rows: rows}); err != nil {
		slog.Warn("ws write initial links", "err", err)
		return err
	}
	return nil
}

func (s *Server) handleAdminPage(w http.ResponseWriter, r *http.Request) {
	if !s.basicAuth(w, r) {
		w.Header().Set("WWW-Authenticate", `Basic realm="admin"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := view.AdminTmpl.Execute(w, s.view.BuildHTMLContext()); err != nil {
		slog.Error("execute admin tpl", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
