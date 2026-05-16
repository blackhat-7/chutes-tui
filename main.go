package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/blackhat-7/chutes-tui/cmd"
	"github.com/blackhat-7/chutes-tui/internal/api"
)

func main() {
	api.LoadEnv(".env")

	app := cmd.NewApp()
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
