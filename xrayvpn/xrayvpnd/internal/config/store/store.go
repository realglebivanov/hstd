package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
)

var (
	mu                     sync.Mutex
	ErrAlreadyInitialized = errors.New("links already initialized")
)

const statePath = "/etc/xrayvpn/state.json"

func GetState() (*State, error) {
	mu.Lock()
	defer mu.Unlock()
	return loadState()
}

func AddLink(link string, rotate bool) error {
	mu.Lock()
	defer mu.Unlock()

	st, err := loadState()
	if err != nil {
		return err
	}

	if err := st.addLink(link, rotate); err != nil {
		return err
	}

	return saveState(st)
}

func RemoveLink(id string) (activeChanged bool, err error) {
	mu.Lock()
	defer mu.Unlock()

	st, err := loadState()
	if err != nil {
		return false, err
	}

	wasActive, err := st.removeLink(id)
	if err != nil {
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

func InitLinks(serverLink, proxyLink string) error {
	mu.Lock()
	defer mu.Unlock()

	st, err := loadState()
	if err != nil {
		return err
	}
	if len(st.Links) > 0 {
		return ErrAlreadyInitialized
	}

	if err := st.addLink(proxyLink, true); err != nil {
		return fmt.Errorf("add proxy link: %w", err)
	}
	if err := st.addLink(serverLink, true); err != nil {
		return fmt.Errorf("add server link: %w", err)
	}
	return saveState(st)
}

func RotateUUID(uuid string) error {
	mu.Lock()
	defer mu.Unlock()

	st, err := loadState()
	if err != nil {
		return err
	}

	if err := st.rotateUUID(uuid); err != nil {
		return err
	}

	return saveState(st)
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

	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o660)
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
