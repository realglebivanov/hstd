package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/xtls/xray-core/infra/conf"
)

type Link struct {
	ID   string `json:"id"`
	Link string `json:"link"`
}

func (l Link) Summary() string {
	u, err := url.Parse(l.Link)
	if err != nil {
		return l.Link
	}
	uuid := u.User.Username()
	if len(uuid) > 13 {
		uuid = uuid[:13]
	}
	return uuid + "@" + u.Host
}

type State struct {
	Links    []Link `json:"links"`
	ActiveID string `json:"active_id"`
}

var mu sync.Mutex

const statePath = "/etc/xray-vpn/state.json"

func GetState() (*State, error) {
	mu.Lock()
	defer mu.Unlock()
	return loadState()
}

func AddLink(link string) error {
	mu.Lock()
	defer mu.Unlock()

	link = strings.TrimSpace(link)
	if _, err := parseLink(link); err != nil {
		return fmt.Errorf("Invalid link: %v", err)
	}

	st, err := loadState()
	if err != nil {
		return err
	}

	if err := st.addLink(link); err != nil {
		return err
	}

	return saveState(st)
}

func RemoveLink(id string) (activeChanged bool, err error) {
	mu.Lock()
	defer mu.Unlock()

	id = strings.TrimSpace(id)

	st, err := loadState()
	if err != nil {
		return false, err
	}

	wasActive := st.ActiveID == id
	if err := st.removeLink(id); err != nil {
		return false, err
	}

	return wasActive, saveState(st)
}

func ChooseLink(id string) error {
	mu.Lock()
	defer mu.Unlock()

	st, err := loadState()
	if err != nil {
		return err
	}

	if err := st.chooseLink(id); err != nil {
		return err
	}

	return saveState(st)
}

func GetActiveOutboundConfig() (*conf.OutboundDetourConfig, error) {
	mu.Lock()
	defer mu.Unlock()

	st, err := loadState()
	if err != nil {
		return nil, err
	}

	return st.activeOutboundConfig()
}

func loadState() (*State, error) {
	data, err := os.ReadFile(statePath)
	if errors.Is(err, os.ErrNotExist) {
		return &State{Links: []Link{}, ActiveID: ""}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read state file: %w", err)
	}
	if len(data) == 0 {
		return &State{Links: []Link{}, ActiveID: ""}, nil
	}

	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("unmarshal state file: %w", err)
	}
	return &st, nil
}

func saveState(st *State) error {
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	data = append(data, '\n')
	tmpPath := statePath + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0o660); err != nil {
		return fmt.Errorf("write temp state file: %w", err)
	}

	if err := os.Rename(tmpPath, statePath); err != nil {
		return fmt.Errorf("rename temp state file: %w", err)
	}
	_ = os.Remove(tmpPath)

	return nil
}
