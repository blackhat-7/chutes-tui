package api

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/blackhat-7/chutes-tui/internal/model"
)

const chutesURL = "https://chutes.ai/app/research/utilization"

var (
	objRe = regexp.MustCompile(`\{chute_id:[^}]+\}`)
	kvRe  = regexp.MustCompile(`(\w+):("(?:[^"\\]|\\.)*"|-?\.?\d[\d.]*(?:[eE][+-]?\d+)?|true|false|null)`)
)

func FetchChutes() ([]model.Chute, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", chutesURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (chutes-tui/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parseHTML(string(body)), nil
}

func parseHTML(html string) []model.Chute {
	var chutes []model.Chute
	seen := make(map[string]struct{})

	for _, match := range objRe.FindAllString(html, -1) {
		p := parseObj(match)
		cid, _ := p["chute_id"].(string)
		name, _ := p["name"].(string)
		if cid == "" || name == "" || strings.Contains(strings.ToLower(name), "[private") {
			continue
		}
		if _, ok := seen[cid]; ok {
			continue
		}
		seen[cid] = struct{}{}

		chutes = append(chutes, model.Chute{
			ChuteID:               cid,
			Name:                  name,
			UtilizationCurrent:    toFloat(p["utilization_current"]),
			Utilization5m:         toFloat(p["utilization_5m"]),
			Utilization1h:         toFloat(p["utilization_1h"]),
			RateLimitRatio5m:      toFloat(p["rate_limit_ratio_5m"]),
			RateLimitRatio1h:      toFloat(p["rate_limit_ratio_1h"]),
			TotalRequests1h:       toFloat(p["total_requests_1h"]),
			CompletedRequests1h:   toFloat(p["completed_requests_1h"]),
			RateLimitedRequests1h: toFloat(p["rate_limited_requests_1h"]),
			InstanceCount:         toInt(p["instance_count"]),
			ActiveInstanceCount:   toInt(p["active_instance_count"]),
			Scalable:              toBool(p["scalable"]),
			ActionTaken:           toString(p["action_taken"]),
			EffectiveMultiplier:   toFloat(p["effective_multiplier"]),
			TEE:                   toBool(p["tee"]),
		})
	}

	return chutes
}

func parseObj(s string) map[string]any {
	out := make(map[string]any)
	for _, m := range kvRe.FindAllStringSubmatch(s, -1) {
		k, v := m[1], m[2]
		switch {
		case strings.HasPrefix(v, `"`):
			out[k] = strings.ReplaceAll(v[1:len(v)-1], `\"`, `"`)
		case v == "true":
			out[k] = true
		case v == "false":
			out[k] = false
		case v == "null":
			out[k] = nil
		default:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				out[k] = f
			} else {
				out[k] = v
			}
		}
	}
	return out
}

func toFloat(v any) float64 {
	if v == nil {
		return 0
	}
	if f, ok := v.(float64); ok {
		return f
	}
	if s, ok := v.(string); ok {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}
	return 0
}

func toInt(v any) int {
	if v == nil {
		return 0
	}
	if f, ok := v.(float64); ok {
		return int(f)
	}
	if s, ok := v.(string); ok {
		i, _ := strconv.Atoi(s)
		return i
	}
	return 0
}

func toBool(v any) bool {
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	if s, ok := v.(string); ok {
		return s == "true"
	}
	return false
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	if f, ok := v.(float64); ok {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	return fmt.Sprintf("%v", v)
}
