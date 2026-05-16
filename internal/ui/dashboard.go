package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/blackhat-7/chutes-tui/internal/model"
)

func DashboardView(chutes []model.Chute, usage map[string]any, apiKeySet bool, width int) string {
	var b strings.Builder

	b.WriteString(UsageBar(usage, apiKeySet))
	b.WriteString("\n\n")
	b.WriteString(StatsBanner(chutes))
	b.WriteString("\n\n")

	sorted := make([]model.Chute, len(chutes))
	copy(sorted, chutes)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].UtilizationCurrent > sorted[j].UtilizationCurrent
	})

	rows := make([][]string, 0, len(sorted))
	for _, c := range sorted {
		name := c.DisplayName()
		if c.TEE {
			name = BrightCyanStyle.Render("[TEE]") + " " + lipgloss.NewStyle().Bold(true).Render(name)
		}

		ai, ti := c.ActiveInstanceCount, c.InstanceCount
		instStr := fmt.Sprintf("%d/%d", ai, ti)
		var instStyled string
		switch {
		case ai == 0:
			instStyled = DimStyle.Render(instStr)
		case ai < ti:
			instStyled = WarningStyle.Render(instStr)
		default:
			instStyled = SuccessStyle.Render(instStr)
		}

		action := TitleCase(strings.ReplaceAll(c.ActionTaken, "_", " "))
		var actionStyled string
		switch {
		case strings.Contains(action, "Scale Up"):
			actionStyled = "^ " + WarningStyle.Render(action)
		case strings.Contains(action, "Scale Down"):
			actionStyled = "v " + CyanStyle.Render(action)
		default:
			actionStyled = DimStyle.Render(action)
		}

		rows = append(rows, []string{
			name,
			Pct(c.UtilizationCurrent, false),
			Pct(c.Utilization5m, false),
			Pct(c.Utilization1h, false),
			Pct(c.RateLimitRatio1h, false),
			instStyled,
			FmtRequests(c.TotalRequests1h),
			actionStyled,
		})
	}

	// Build table
	headers := []string{"Model", "Cur %", "5m %", "1h %", "RL %", "Instances", "Req / 1h", "Action"}
	colWidths := []int{38, 8, 8, 8, 8, 11, 11, 20}

	b.WriteString(renderTable(headers, rows, colWidths, width))
	return b.String()
}
