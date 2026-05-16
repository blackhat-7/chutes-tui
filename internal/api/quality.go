package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/blackhat-7/chutes-tui/internal/model"
)

func FetchQualityScores(apiKey string) (map[string]int, string, error) {
	if apiKey == "" {
		return nil, "fallback scores (set ARTIFICIAL_ANALYSIS_API_KEY for live)", nil
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", "https://artificialanalysis.ai/api/v2/data/llms/models", nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("x-api-key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Sprintf("quality fetch failed (%T) — using fallback", err), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Sprintf("quality API %d — using fallback", resp.StatusCode), nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Sprintf("quality body read failed — using fallback"), nil
	}

	var payload struct {
		Data []struct {
			Slug         string `json:"slug"`
			Name         string `json:"name"`
			Evaluations  struct {
				Index *float64 `json:"artificial_analysis_intelligence_index"`
			} `json:"evaluations"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, "quality JSON parse failed — using fallback", nil
	}

	type pair struct {
		key   string
		score float64
	}
	var raw []pair
	for _, m := range payload.Data {
		if m.Evaluations.Index == nil {
			continue
		}
		score := *m.Evaluations.Index
		if m.Slug != "" {
			raw = append(raw, pair{model.Norm(m.Slug), score})
		}
		if m.Name != "" {
			raw = append(raw, pair{model.Norm(m.Name), score})
		}
	}

	if len(raw) == 0 {
		return nil, "quality API returned no scores — using fallback", nil
	}

	lo, hi := raw[0].score, raw[0].score
	for _, p := range raw[1:] {
		if p.score < lo {
			lo = p.score
		}
		if p.score > hi {
			hi = p.score
		}
	}
	out := make(map[string]int, len(raw))
	if hi == lo {
		for _, p := range raw {
			out[p.key] = 100
		}
	} else {
		span := hi - lo
		for _, p := range raw {
			out[p.key] = int((p.score - lo) / span * 100)
		}
	}

	return out, fmt.Sprintf("live scores (%d models)", len(out)), nil
}
