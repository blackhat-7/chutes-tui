package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const blockFull = '█'
const blockEmpty = '░'

func Pct(value float64, higherGood bool) string {
	pct := value * 100
	var col lipgloss.Style
	if higherGood {
		if pct > 70 {
			col = SuccessStyle
		} else if pct > 40 {
			col = WarningStyle
		} else {
			col = DangerStyle
		}
	} else {
		if pct > 70 {
			col = DangerStyle
		} else if pct > 30 {
			col = WarningStyle
		} else {
			col = SuccessStyle
		}
	}
	return col.Render(fmt.Sprintf("%5.1f%%", pct))
}

func ScoreBar(score float64, width int) string {
	filled := int(math.Round(score * float64(width)))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	bar := strings.Repeat(string(blockFull), filled) + strings.Repeat(string(blockEmpty), width-filled)
	var col lipgloss.Style
	if score > 0.7 {
		col = SuccessStyle
	} else if score > 0.45 {
		col = WarningStyle
	} else {
		col = DangerStyle
	}
	return col.Render(fmt.Sprintf("%s %4.1f", bar, score*100))
}

func FmtRequests(n float64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", n/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", n/1_000)
	}
	return fmt.Sprintf("%.0f", n)
}

func QualCell(q int, hasCache bool) string {
	if q == 50 && hasCache {
		return DimStyle.Render("?")
	}
	var col lipgloss.Style
	if q >= 85 {
		col = SuccessStyle
	} else if q >= 72 {
		col = WarningStyle
	} else {
		col = DangerStyle
	}
	return col.Render(fmt.Sprintf("%d", q))
}

func RankStyle(rank int, rl bool) lipgloss.Style {
	if rl {
		return DimStyle
	}
	switch {
	case rank == 1:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("226"))
	case rank <= 3:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("82"))
	case rank <= 10:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220"))
	default:
		return DimStyle
	}
}

func TitleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

func NameStyle(tee bool) lipgloss.Style {
	if tee {
		return lipgloss.NewStyle().Bold(true)
	}
	return lipgloss.NewStyle()
}
