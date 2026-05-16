package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/blackhat-7/chutes-tui/internal/model"
)

func StatsBanner(chutes []model.Chute) string {
	active := 0
	instances := 0
	reqs := 0.0
	ratelim := 0.0
	scalable := 0

	for _, c := range chutes {
		if c.ActiveInstanceCount > 0 {
			active++
		}
		instances += c.InstanceCount
		reqs += c.TotalRequests1h
		ratelim += c.RateLimitedRequests1h
		if c.Scalable {
			scalable++
		}
	}

	cards := []string{
		card(fmt.Sprintf("%d", active), "Active Models", CyanStyle),
		card(fmt.Sprintf("%d", instances), "Instances", CyanStyle),
		card(fmt.Sprintf("%.1fK", reqs/1000), "Req / 1h", CyanStyle),
		card(fmt.Sprintf("%.1fK", ratelim/1000), "Rate-Lim / 1h", WarningStyle),
		card(fmt.Sprintf("%d", scalable), "Scalable", SuccessStyle),
	}

	return BorderedPanel.Render(strings.Join(cards, "  "))
}

func card(value, label string, valStyle lipgloss.Style) string {
	return valStyle.Render(value) + "\n" + DimStyle.Render(label)
}
