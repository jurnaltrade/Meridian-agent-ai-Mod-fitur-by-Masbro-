package persistence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"meridian-go-rewrite/internal/config"
)

type PositionState struct {
	Closed     bool   `json:"closed"`
	DeployedAt string `json:"deployed_at"`
	ClosedAt   string `json:"closed_at"`
}

type StateData struct {
	Positions    map[string]PositionState `json:"positions"`
	RecentEvents []interface{}            `json:"recentEvents"`
}

var (
	stateMutex sync.RWMutex
	stateFile  string
)

func initStatePath() {
	if stateFile == "" {
		cfg := config.Get()
		if cfg != nil {
			stateFile = cfg.DataPath("state.json")
		} else {
			stateFile = "state.json"
		}
	}
}

func GetState() StateData {
	initStatePath()
	stateMutex.RLock()
	defer stateMutex.RUnlock()

	data, err := os.ReadFile(stateFile)
	if err != nil {
		return StateData{Positions: make(map[string]PositionState), RecentEvents: []interface{}{}}
	}

	var sd StateData
	if err := json.Unmarshal(data, &sd); err != nil {
		return StateData{Positions: make(map[string]PositionState), RecentEvents: []interface{}{}}
	}
	if sd.Positions == nil {
		sd.Positions = make(map[string]PositionState)
	}
	return sd
}

func SaveState(sd StateData) {
	initStatePath()
	stateMutex.Lock()
	defer stateMutex.Unlock()

	if dir := filepath.Dir(stateFile); dir != "." {
		os.MkdirAll(dir, 0755)
	}
	bytes, _ := json.MarshalIndent(sd, "", "  ")
	os.WriteFile(stateFile, bytes, 0644)
}
