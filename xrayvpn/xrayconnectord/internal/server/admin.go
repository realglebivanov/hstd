package server

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/server/admin/view"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/server/wsconn"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/server/wsconn/state"
)

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

	for {
		evt, err := wsc.ReadEvent()
		if errors.Is(err, state.ErrConnClosed) {
			slog.Info("ws conn closed")
			break
		}
		if evtErr, ok := errors.AsType[*wsconn.ParseError](err); ok {
			slog.Info("bad ws event", "err", err)
			wsc.Send(wsconn.ErrorMsg{Type: "error", Message: evtErr.Message})
			continue
		}
		if err != nil {
			slog.Warn("read ws event", "err", err)
			break
		}

		switch req := evt.(type) {
		case *wsconn.UpdateLinkReq:
			s.handleUpdateLink(wsc, req)
		}
	}
}

func (s *Server) handleUpdateLink(wsc *wsconn.WSConn, req *wsconn.UpdateLinkReq) {
	l, err := s.db.UpdateLink(req.Index, req.Comment, req.Enabled)
	if err != nil {
		slog.Error("ws update link", "err", err)
		wsc.Send(wsconn.ErrorMsg{Type: "error", Message: "internal error"})
		return
	}

	row, err := s.view.BuildRow(l)
	if err != nil {
		slog.Error("ws build row", "err", err)
		wsc.Send(wsconn.ErrorMsg{Type: "error", Message: "internal error"})
		return
	}

	wsc.Send(wsconn.LinkUpdatedMsg{Type: "link_updated", Row: row})
	s.broadcast.Broadcast(row, wsc)
}

func (s *Server) sendLinks(wsc *wsconn.WSConn) error {
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
	if err := wsc.Send(wsconn.LinksMsg{Type: "links", Rows: rows}); err != nil {
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

	var buf bytes.Buffer
	if err := view.AdminTmpl.Execute(&buf, s.view.BuildHTMLContext()); err != nil {
		slog.Error("execute admin tpl", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
}
