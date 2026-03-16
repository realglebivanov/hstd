package store

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

type State struct {
	Links    []Link `json:"links"`
	ActiveID string `json:"active_id"`
}

func (s *State) addLink(link string) error {
	link = strings.TrimSpace(link)

	if err := validateLink(link); err != nil {
		return fmt.Errorf("Invalid link: %v", err)
	}

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

func hashID(link string) string {
	h := sha256.Sum256([]byte(link))
	return hex.EncodeToString(h[:4])
}
