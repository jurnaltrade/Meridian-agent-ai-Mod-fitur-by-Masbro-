package agentmeridian

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const DefaultBase = "https://api.agentmeridian.xyz"

var httpClient = &http.Client{Timeout: 30 * time.Second}

func AgentMeridianJSON(method, path string, body any) ([]byte, error) {
	base := os.Getenv("AGENT_MERIDIAN_BASE")
	if base == "" {
		base = DefaultBase
	}
	agentID := os.Getenv("AGENT_MERIDIAN_ID")
	if agentID == "" {
		agentID = os.Getenv("AGENT_ID")
	}

	url := base + path

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if agentID != "" {
		req.Header.Set("x-agent-id", agentID)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("agent meridian request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		limit := len(data)
		if limit > 200 {
			limit = 200
		}
		return nil, fmt.Errorf("agent meridian HTTP %d: %s", resp.StatusCode, string(data[:limit]))
	}
	return data, nil
}

func RelayDeploy(payload map[string]any) (string, error) {
	data, err := AgentMeridianJSON("POST", "/deploy-position", payload)
	if err != nil {
		return "", err
	}
	var result struct {
		Success  bool   `json:"success"`
		Tx       string `json:"tx"`
		Position string `json:"position"`
		Error    string `json:"error"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("parse deploy response: %w", err)
	}
	if !result.Success {
		return "", fmt.Errorf("deploy failed: %s", result.Error)
	}
	return string(data), nil
}

func RelayClose(payload map[string]any) (string, error) {
	data, err := AgentMeridianJSON("POST", "/close-position", payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func RelayClaim(payload map[string]any) (string, error) {
	data, err := AgentMeridianJSON("POST", "/claim-fees", payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
