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

func (s *State) replaceDefaultLinks(serverLink, proxyLink string) error {
	var staleIds []string

	for _, link := range s.Links {
		if !link.Rotate {
			continue
		}
		staleIds = append(staleIds, link.ID)
	}

	for _, ID := range staleIds {
		if _, err := s.removeLink(ID); err != nil {
			return err
		}
	}

	return errors.Join(s.addLink(proxyLink, true), s.addLink(serverLink, true))
}

func (s *State) addLink(link string, rotate bool) error {
	link = strings.TrimSpace(link)

	if err := validateLink(link); err != nil {
		return fmt.Errorf("invalid link: %v", err)
	}

	for _, existing := range s.Links {
		if existing.Link == link {
			return fmt.Errorf("link already exists")
		}
	}

	id := hashID(link)
	s.ActiveID = id
	s.Links = append(s.Links, Link{ID: id, Link: link, Rotate: rotate})

	return nil
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
