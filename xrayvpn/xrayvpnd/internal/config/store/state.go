package store

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/realglebivanov/hstd/hstdlib/xrayconf"
)

type State struct {
	Links    []Link `json:"links"`
	ActiveID string `json:"active_id"`
}

func (s *State) getActiveLink() (string, error) {
	if s.ActiveID == "" {
		return "", fmt.Errorf("no active link selected")
	}

	for _, item := range s.Links {
		if item.ID == s.ActiveID {
			return item.Link, nil
		}
	}

	return "", fmt.Errorf("active link %q not found in state", s.ActiveID)
}

func (s *State) replaceDefaultLinks(serverLink, proxyLink string) error {
	s.Links = slices.DeleteFunc(s.Links, func(l Link) bool {
		return l.Rotate
	})

	activeID, pErr := s.addLink(proxyLink, true)
	_, sErr := s.addLink(serverLink, true)

	if err := errors.Join(pErr, sErr); err != nil {
		return err
	}

	s.ActiveID = activeID

	return nil
}

func (s *State) addLink(link string, rotate bool) (string, error) {
	link = strings.TrimSpace(link)

	if _, err := xrayconf.ParseVLESSLink(link); err != nil {
		return "", fmt.Errorf("invalid link: %v", err)
	}

	for _, existing := range s.Links {
		if existing.Link == link {
			return "", fmt.Errorf("link already exists")
		}
	}

	id := hashID(link)
	s.Links = append(s.Links, Link{ID: id, Link: link, Rotate: rotate})

	return id, nil
}

func (s *State) removeLink(id string) (bool, error) {
	id = strings.TrimSpace(id)
	wasActive := id == s.ActiveID

	idx := -1
	for i, item := range s.Links {
		if item.ID == id {
			idx = i
			break
		}
	}

	if idx == -1 {
		return wasActive, fmt.Errorf("link id %q not found", id)
	}

	s.Links = append(s.Links[:idx], s.Links[idx+1:]...)

	if s.ActiveID == id {
		if len(s.Links) > 0 {
			s.ActiveID = s.Links[0].ID
		} else {
			s.ActiveID = ""
		}
	}
	return wasActive, nil
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

func hashID(link string) string {
	h := sha256.Sum256([]byte(link))
	return hex.EncodeToString(h[:4])
}
