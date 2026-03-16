package store

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/xtls/xray-core/infra/conf"
)

func (s *State) addLink(link string) error {
	for _, existing := range s.Links {
		if existing.Link == link {
			return fmt.Errorf("link already exists")
		}
	}

	id := hashID(link)
	s.ActiveID = id
	s.Links = append(s.Links, Link{ID: id, Link: link})

	return nil
}

func (s *State) removeLink(id string) error {
	id = strings.TrimSpace(id)

	idx := -1
	for i, item := range s.Links {
		if item.ID == id {
			idx = i
			break
		}
	}

	if idx == -1 {
		return fmt.Errorf("link id %q not found", id)
	}

	s.Links = append(s.Links[:idx], s.Links[idx+1:]...)

	if s.ActiveID == id {
		if len(s.Links) > 0 {
			s.ActiveID = s.Links[0].ID
		} else {
			s.ActiveID = ""
		}
	}
	return nil
}

func (s *State) chooseLink(id string) error {
	id = strings.TrimSpace(id)

	if id == "" {
		return errors.New("empty id")
	}

	for _, item := range s.Links {
		if item.ID == id {
			s.ActiveID = id
			return nil
		}
	}
	return fmt.Errorf("link id %q not found", id)
}

func (s *State) activeOutboundConfig() (*conf.OutboundDetourConfig, error) {
	if s.ActiveID == "" {
		return nil, fmt.Errorf("no active link selected")
	}

	var raw string
	for _, item := range s.Links {
		if item.ID == s.ActiveID {
			raw = item.Link
			break
		}
	}

	if raw == "" {
		return nil, fmt.Errorf("active link %q not found in state", s.ActiveID)
	}

	out, err := parseLink(raw)
	if err != nil {
		return nil, fmt.Errorf("parse link: %w", err)
	}

	return out, nil
}

func hashID(link string) string {
	h := sha256.Sum256([]byte(link))
	return hex.EncodeToString(h[:4])
}
