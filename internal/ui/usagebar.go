package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func UsageBar(usage map[string]any, apiKeySet bool) string {
	if !apiKeySet {
		return DimStyle.Render("  My Usage  ·  set CHUTES_API_KEY to see personal usage")
	}
	if len(usage) == 0 {
		return DimStyle.Render("  My Usage  ·  loading...")
	}

	var parts []string
	parts = append(parts, TitleStyle.Render("My Usage"))

	for key, val := range usage {
		label := TitleCase(strings.ReplaceAll(key, "_", " "))
		switch v := val.(type) {
		case map[string]any:
			used := 0.0
			cap := 0.0
			if u, ok := v["used"].(float64); ok {
				used = u
			} else if u, ok := v["usage"].(float64); ok {
				used = u
			}
			if c, ok := v["cap"].(float64); ok {
				cap = c
			} else if c, ok := v["limit"].(float64); ok {
				cap = c
			}
			if cap > 0 {
				ratio := used / cap
				clamped := math.Min(ratio, 1.0)
				var col lipgloss.Style
				if ratio > 0.85 {
					col = DangerStyle
				} else if ratio > 0.6 {
					col = WarningStyle
				} else {
					col = SuccessStyle
				}
				filled := int(math.Round(clamped * 10))
				bar := strings.Repeat(string(blockFull), filled) + strings.Repeat(string(blockEmpty), 10-filled)
				var pctStr string
				if ratio > 1.0 {
					pctStr = fmt.Sprintf("OVER (%.0f%%)", ratio*100)
				} else {
					pctStr = fmt.Sprintf("%.0f%%", ratio*100)
				}
				parts = append(parts, DimStyle.Render(label+":")+" "+col.Render(bar+" "+pctStr))
			} else if used > 0 {
				parts = append(parts, DimStyle.Render(label+":")+" "+CyanStyle.Render(fmt.Sprintf("%.0f", used)))
			}
		case float64:
			if v > 0 {
				parts = append(parts, DimStyle.Render(label+":")+" "+CyanStyle.Render(fmt.Sprintf("%.2f", v)))
			}
		case int:
			if v > 0 {
				parts = append(parts, DimStyle.Render(label+":")+" "+CyanStyle.Render(fmt.Sprintf("%d", v)))
			}
		}
	}

	return "  ·  " + strings.Join(parts[1:], "  ·  ")
}
