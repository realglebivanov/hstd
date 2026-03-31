package wsconn

import (
	"encoding/json"
	"fmt"

	"github.com/realglebivanov/hstd/xrayconnectord/internal/server/admin/view"
)

type LinksMsg struct {
	Type string      `json:"type"`
	Rows []*view.Row `json:"rows"`
}

type LinkUpdatedMsg struct {
	Type string    `json:"type"`
	Row  *view.Row `json:"row"`
}

type ErrorMsg struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type UpdateLinkReq struct {
	Index   int     `json:"index"`
	Comment *string `json:"comment"`
	Enabled *bool   `json:"enabled"`
}

type ParseError struct {
	Message string
}

func (e *ParseError) Error() string { return e.Message }

func parseEvent(data []byte) (any, error) {
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, &ParseError{Message: "invalid json"}
	}

	var v any
	switch envelope.Type {
	case "update_link":
		v = &UpdateLinkReq{}
	default:
		return nil, &ParseError{Message: fmt.Sprintf("unknown type: %s", envelope.Type)}
	}

	if err := json.Unmarshal(data, v); err != nil {
		return nil, &ParseError{Message: fmt.Sprintf("invalid json for %s", envelope.Type)}
	}
	return v, nil
}
