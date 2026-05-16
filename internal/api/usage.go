package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const chutesAPIBase = "https://api.chutes.ai"

func FetchUsage(apiKey string) (map[string]any, string, error) {
	if apiKey == "" {
		return nil, "", nil
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", chutesAPIBase+"/users/me/subscription_usage", nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Sprintf("usage fetch failed: %v", err), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Sprintf("usage API %d", resp.StatusCode), nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "usage body read failed", nil
	}

	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, "usage JSON parse failed", nil
	}

	return out, "", nil
}

func LoadEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			if os.Getenv(key) == "" {
				os.Setenv(key, val)
			}
		}
	}
}
