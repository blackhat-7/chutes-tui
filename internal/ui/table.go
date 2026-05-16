package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderTable(headers []string, rows [][]string, colWidths []int, maxWidth int) string {
	if len(headers) != len(colWidths) {
		return ""
	}

	// Calculate total width needed
	totalWidth := 0
	for _, w := range colWidths {
		totalWidth += w + 2 // +2 for padding
	}
	totalWidth-- // last column doesn't need trailing space

	// Scale down columns if terminal is too narrow
	scale := 1.0
	if maxWidth > 0 && totalWidth > maxWidth {
		scale = float64(maxWidth) / float64(totalWidth)
	}

	var scaledWidths []int
	for _, w := range colWidths {
		scaledWidths = append(scaledWidths, int(float64(w)*scale))
	}

	var b strings.Builder

	// Header
	for i, h := range headers {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(lipgloss.NewStyle().Bold(true).Render(padRight(truncate(stripANSI(h), scaledWidths[i]), scaledWidths[i])))
	}
	b.WriteString("\n")

	// Separator
	for i := range headers {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(strings.Repeat("─", scaledWidths[i]))
	}
	b.WriteString("\n")

	// Rows
	for rowIdx, row := range rows {
		zebra := rowIdx%2 == 1
		for i, cell := range row {
			if i > 0 {
				b.WriteString("  ")
			}
			truncated := truncate(cell, scaledWidths[i])
			actualLen := visibleLength(truncated)
			padding := scaledWidths[i] - actualLen
			if padding < 0 {
				padding = 0
			}
			rendered := truncated + strings.Repeat(" ", padding)
			if zebra {
				rendered = lipgloss.NewStyle().Background(lipgloss.Color("236")).Render(rendered)
			}
			b.WriteString(rendered)
		}
		b.WriteString("\n")
	}

	return b.String()
}

func padRight(s string, width int) string {
	visLen := visibleLength(s)
	if visLen >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visLen)
}

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	visLen := visibleLength(s)
	if visLen <= maxLen {
		return s
	}

	// Need to truncate, preserving ANSI codes if present
	var result strings.Builder
	currentLen := 0
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			result.WriteRune(r)
			continue
		}
		if inEscape {
			result.WriteRune(r)
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		if currentLen >= maxLen-1 {
			result.WriteRune('…')
			break
		}
		result.WriteRune(r)
		currentLen++
	}
	return result.String()
}

func visibleLength(s string) int {
	length := 0
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		length++
	}
	return length
}

func stripANSI(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}
