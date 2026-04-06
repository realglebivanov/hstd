package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/realglebivanov/hstd/hstdlib/xrayconf"
)

type State struct {
	Subs     []*Sub  `json:"subs,omitempty"`
	Conns    []*Conn `json:"conns"`
	ActiveID string  `json:"active_id"`
}

func (s *State) getActiveConfig() (*xrayconf.Config, error) {
	if s.ActiveID == "" {
		return nil, fmt.Errorf("no active conn selected")
	}

	for _, item := range s.Conns {
		if item.ID == s.ActiveID {
			var cfg xrayconf.Config
			if err := json.Unmarshal(item.Config, &cfg); err != nil {
				return nil, fmt.Errorf("unmarshal config: %w", err)
			}
			return &cfg, nil
		}
	}

	return nil, fmt.Errorf("active conn %q not found in state", s.ActiveID)
}

func (s *State) syncConns(cfgs map[string][]*xrayconf.Config) error {
	s.Conns = slices.DeleteFunc(s.Conns, func(c *Conn) bool {
		_, synced := cfgs[c.SubID]
		return synced
	})

	var errs []error
	var activeIDMet bool

	for subID, configs := range cfgs {
		for _, cfg := range configs {
			conn, err := s.addConn(cfg, subID)
			activeIDMet = activeIDMet || conn.ID == s.ActiveID
			errs = append(errs, err)
		}
	}

	if !activeIDMet {
		s.ActiveID = ""
		if len(s.Conns) > 0 {
			s.ActiveID = s.Conns[0].ID
		}
	}

	return errors.Join(errs...)
}

func (s *State) addSub(url string) {
	sub := NewSub(url)
	if slices.ContainsFunc(s.Subs, func(s *Sub) bool { return s.ID == sub.ID }) {
		return
	}
	s.Subs = append(s.Subs, sub)
}

func (s *State) removeSub(id string) error {
	idx := slices.IndexFunc(s.Subs, func(s *Sub) bool { return s.ID == id })
	if idx == -1 {
		return fmt.Errorf("subscription %q not found", id)
	}

	subID := s.Subs[idx].ID
	s.Subs = slices.Delete(s.Subs, idx, idx+1)
	s.Conns = slices.DeleteFunc(s.Conns, func(c *Conn) bool { return c.SubID == subID })

	if s.ActiveID == "" {
		return nil
	}
	if slices.ContainsFunc(s.Conns, func(c *Conn) bool { return c.ID == s.ActiveID }) {
		return nil
	}

	s.ActiveID = ""
	if len(s.Conns) > 0 {
		s.ActiveID = s.Conns[0].ID
	}
	return nil
}

func (s *State) addConn(cfg *xrayconf.Config, subID string) (*Conn, error) {
	conn, err := NewConn(cfg, subID)
	if err != nil {
		return nil, fmt.Errorf("failed to build a conn")
	}

	for _, existing := range s.Conns {
		if existing.ID == conn.ID {
			return nil, fmt.Errorf("conn %q already exists", conn.Remark)
		}
	}

	s.Conns = append(s.Conns, conn)

	return conn, nil
}

func (s *State) removeConn(id string) (bool, error) {
	id = strings.TrimSpace(id)
	wasActive := id == s.ActiveID

	idx := -1
	for i, item := range s.Conns {
		if item.ID == id {
			idx = i
			break
		}
	}

	if idx == -1 {
		return wasActive, fmt.Errorf("conn id %q not found", id)
	}

	s.Conns = append(s.Conns[:idx], s.Conns[idx+1:]...)

	if s.ActiveID == id {
		s.ActiveID = ""
		if len(s.Conns) > 0 {
			s.ActiveID = s.Conns[0].ID
		}
	}
	return wasActive, nil
}

func (s *State) chooseConn(id string) error {
	id = strings.TrimSpace(id)

	if id == "" {
		return errors.New("empty id")
	}

	for _, item := range s.Conns {
		if item.ID == id {
			s.ActiveID = id
			return nil
		}
	}
	return fmt.Errorf("conn id %q not found", id)
}
