package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/realglebivanov/hstd/hstdlib/xrayconf"
)

var mu sync.Mutex

const statePath = "/etc/xrayvpn/state.json"

func GetState() (*State, error) {
	mu.Lock()
	defer mu.Unlock()
	return loadState()
}

func GetActiveConfig() (*xrayconf.Config, error) {
	mu.Lock()
	defer mu.Unlock()

	st, err := loadState()
	if err != nil {
		return nil, err
	}

	return st.getActiveConfig()
}

func AddConn(cfg *xrayconf.Config) error {
	mu.Lock()
	defer mu.Unlock()

	st, err := loadState()
	if err != nil {
		return err
	}

	if _, err := st.addConn(cfg, ""); err != nil {
		return err
	}

	return saveState(st)
}

func SyncConns(cfgs map[string][]*xrayconf.Config) error {
	mu.Lock()
	defer mu.Unlock()

	st, err := loadState()
	if err != nil {
		return err
	}
	if err := st.syncConns(cfgs); err != nil {
		return fmt.Errorf("sync conns: %w", err)
	}

	return saveState(st)
}

func GetSubs() ([]*Sub, error) {
	mu.Lock()
	defer mu.Unlock()

	st, err := loadState()
	if err != nil {
		return nil, err
	}
	return st.Subs, nil
}

func AddSub(url string) error {
	mu.Lock()
	defer mu.Unlock()

	st, err := loadState()
	if err != nil {
		return err
	}

	st.addSub(url)

	return saveState(st)
}

func RemoveSub(id string) error {
	mu.Lock()
	defer mu.Unlock()

	st, err := loadState()
	if err != nil {
		return err
	}

	if err := st.removeSub(id); err != nil {
		return err
	}

	return saveState(st)
}

func RemoveConn(id string) (activeChanged bool, err error) {
	mu.Lock()
	defer mu.Unlock()

	st, err := loadState()
	if err != nil {
		return false, err
	}

	wasActive, err := st.removeConn(id)
	if err != nil {
		return false, err
	}

	return wasActive, saveState(st)
}

func ChooseConn(id string) error {
	mu.Lock()
	defer mu.Unlock()

	st, err := loadState()
	if err != nil {
		return err
	}

	if err := st.chooseConn(id); err != nil {
		return err
	}

	return saveState(st)
}

func loadState() (*State, error) {
	data, err := os.ReadFile(statePath)
	if errors.Is(err, os.ErrNotExist) {
		return &State{Conns: []*Conn{}, ActiveID: ""}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read state file: %w", err)
	}
	if len(data) == 0 {
		return &State{Conns: []*Conn{}, ActiveID: ""}, nil
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

	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0660)
	if err != nil {
		return fmt.Errorf("create temp state file: %w", err)
	}
	defer os.Remove(tmpPath)
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("write temp state file: %w", err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("sync temp state file: %w", err)
	}

	return os.Rename(tmpPath, statePath)
}
