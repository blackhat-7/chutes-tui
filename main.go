package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/blackhat-7/chutes-tui/cmd"
	"github.com/blackhat-7/chutes-tui/internal/api"
	"github.com/blackhat-7/chutes-tui/internal/model"
)

func main() {
	api.LoadEnv(".env")

	if handled, err := runCLI(os.Args[1:]); handled {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	app := cmd.NewApp()
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runCLI(args []string) (bool, error) {
	if len(args) == 1 && args[0] == "--best-model-id" {
		return true, printBestModelID()
	}
	return false, nil
}

func printBestModelID() error {
	quality, _, err := api.FetchQualityScores(os.Getenv("ARTIFICIAL_ANALYSIS_API_KEY"))
	if err != nil {
		return err
	}
	if len(quality) > 0 {
		model.SetQualityCache(quality)
	}

	chutes, err := api.FetchChutes()
	if err != nil {
		return err
	}

	best := -1
	for i, c := range chutes {
		if !c.IsUsable() {
			continue
		}
		if best == -1 || c.Score() > chutes[best].Score() || (c.Score() == chutes[best].Score() && c.Name < chutes[best].Name) {
			best = i
		}
	}
	if best == -1 {
		return fmt.Errorf("no usable model found")
	}

	fmt.Println(chutes[best].Name)
	return nil
}
