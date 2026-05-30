package persistence

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"meridian-go-rewrite/internal/config"
	"meridian-go-rewrite/internal/logger"
)

const maxDecisions = 100

type DecisionEntry struct {
	ID       string                 `json:"id"`
	Ts       string                 `json:"ts"`
	Type     string                 `json:"type"`
	Actor    string                 `json:"actor"`
	Pool     *string                `json:"pool"`
	PoolName *string                `json:"pool_name"`
	Position *string                `json:"position"`
	Summary  *string                `json:"summary"`
	Reason   *string                `json:"reason"`
	Risks    []string               `json:"risks"`
	Metrics  map[string]interface{} `json:"metrics"`
	Rejected []string               `json:"rejected"`
}

type DecisionLogData struct {
	Decisions []DecisionEntry `json:"decisions"`
}

var (
	decisionLogMutex sync.RWMutex
	decisionLogFile  string
)

func initDecisionLogPath() {
	if decisionLogFile == "" {
		cfg := config.Get()
		if cfg != nil {
			decisionLogFile = cfg.DataPath("decision-log.json")
		} else {
			decisionLogFile = "decision-log.json"
		}
	}
}

func loadDecisionLog() DecisionLogData {
	initDecisionLogPath()
	decisionLogMutex.RLock()
	defer decisionLogMutex.RUnlock()

	data, err := os.ReadFile(decisionLogFile)
	if err != nil {
		return DecisionLogData{Decisions: []DecisionEntry{}}
	}

	var logData DecisionLogData
	if err := json.Unmarshal(data, &logData); err != nil {
		logger.Error("decision_log_warn", fmt.Errorf("Invalid %s: %w", decisionLogFile, err))
		return DecisionLogData{Decisions: []DecisionEntry{}}
	}
	return logData
}

func saveDecisionLog(data DecisionLogData) {
	initDecisionLogPath()
	decisionLogMutex.Lock()
	defer decisionLogMutex.Unlock()

	if dir := filepath.Dir(decisionLogFile); dir != "." {
		os.MkdirAll(dir, 0755)
	}

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		logger.Error("decision_log_error", fmt.Errorf("Failed to marshal decision log: %w", err))
		return
	}
	os.WriteFile(decisionLogFile, bytes, 0644)
}

func sanitizeStr(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	// Replace all duplicate whitespace
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > maxLen {
		s = s[:maxLen]
	}
	return s
}

func sanitizeStrPtr(s *string, maxLen int) *string {
	if s == nil || *s == "" {
		return nil
	}
	san := sanitizeStr(*s, maxLen)
	if san == "" {
		return nil
	}
	return &san
}

type AppendDecisionInput struct {
	Type     string
	Actor    string
	Pool     *string
	PoolName *string
	Position *string
	Summary  *string
	Reason   *string
	Risks    []string
	Metrics  map[string]interface{}
	Rejected []string
}

func AppendDecision(input AppendDecisionInput) DecisionEntry {
	data := loadDecisionLog()

	randStr := fmt.Sprintf("%06x", rand.Intn(0xffffff))
	id := fmt.Sprintf("dec_%d_%s", time.Now().UnixMilli(), randStr)

	typ := input.Type
	if typ == "" {
		typ = "note"
	}
	actor := input.Actor
	if actor == "" {
		actor = "GENERAL"
	}

	poolName := input.PoolName
	if poolName == nil || *poolName == "" {
		poolName = input.Pool
	}

	var risks []string
	for _, r := range input.Risks {
		if s := sanitizeStr(r, 140); s != "" {
			risks = append(risks, s)
		}
		if len(risks) >= 6 {
			break
		}
	}

	var rejected []string
	for _, r := range input.Rejected {
		if s := sanitizeStr(r, 180); s != "" {
			rejected = append(rejected, s)
		}
		if len(rejected) >= 8 {
			break
		}
	}

	if input.Metrics == nil {
		input.Metrics = make(map[string]interface{})
	}

	entry := DecisionEntry{
		ID:       id,
		Ts:       time.Now().UTC().Format(time.RFC3339),
		Type:     typ,
		Actor:    actor,
		Pool:     input.Pool,
		PoolName: sanitizeStrPtr(poolName, 120),
		Position: input.Position,
		Summary:  sanitizeStrPtr(input.Summary, 280),
		Reason:   sanitizeStrPtr(input.Reason, 500),
		Risks:    risks,
		Metrics:  input.Metrics,
		Rejected: rejected,
	}

	// Prepend entry
	data.Decisions = append([]DecisionEntry{entry}, data.Decisions...)
	if len(data.Decisions) > maxDecisions {
		data.Decisions = data.Decisions[:maxDecisions]
	}

	saveDecisionLog(data)
	return entry
}

func GetRecentDecisions(limit int) []DecisionEntry {
	data := loadDecisionLog()
	if len(data.Decisions) > limit {
		return data.Decisions[:limit]
	}
	return data.Decisions
}

func GetDecisionSummary(limit int) string {
	decisions := GetRecentDecisions(limit)
	if len(decisions) == 0 {
		return "No recent structured decisions yet."
	}

	var lines []string
	for i, d := range decisions {
		pName := "unknown pool"
		if d.PoolName != nil && *d.PoolName != "" {
			pName = *d.PoolName
		} else if d.Pool != nil && *d.Pool != "" {
			pName = *d.Pool
		}

		var bits []string
		bits = append(bits, fmt.Sprintf("%d. [%s] %s %s", i+1, d.Actor, strings.ToUpper(d.Type), pName))
		if d.Summary != nil {
			bits = append(bits, fmt.Sprintf("summary: %s", *d.Summary))
		}
		if d.Reason != nil {
			bits = append(bits, fmt.Sprintf("reason: %s", *d.Reason))
		}
		if len(d.Risks) > 0 {
			bits = append(bits, fmt.Sprintf("risks: %s", strings.Join(d.Risks, ", ")))
		}
		if len(d.Rejected) > 0 {
			bits = append(bits, fmt.Sprintf("rejected: %s", strings.Join(d.Rejected, " | ")))
		}

		lines = append(lines, strings.Join(bits, " | "))
	}
	return strings.Join(lines, "\n")
}
