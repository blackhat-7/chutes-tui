package ui

import "github.com/charmbracelet/lipgloss"

var (
	TitleStyle     = lipgloss.NewStyle().Bold(true)
	DimStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	SuccessStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	WarningStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	DangerStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	CyanStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("51"))
	BrightCyanStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("87"))
	PanelStyle     = lipgloss.NewStyle().Background(lipgloss.Color("235"))
	BorderedPanel  = lipgloss.NewStyle().
						Border(lipgloss.RoundedBorder()).
						BorderForeground(lipgloss.Color("63")).
						Padding(0, 1)
)
