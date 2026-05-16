package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/blackhat-7/chutes-tui/internal/api"
	"github.com/blackhat-7/chutes-tui/internal/model"
	"github.com/blackhat-7/chutes-tui/internal/ui"
)

const refreshInterval = 60 * time.Second

type tab string

const (
	tabDashboard tab = "dashboard"
	tabRankings  tab = "rankings"
)

type appModel struct {
	activeTab   tab
	chutes      []model.Chute
	usage       map[string]any
	qualitySrc  string
	status      string
	width       int
	height      int
	loading     bool
	keys        keyMap
	help        help.Model
	aaAPIKey    string
	chutesKey   string
}

type keyMap struct {
	Refresh key.Binding
	Tab1    key.Binding
	Tab2    key.Binding
	Quit    key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Tab1:    key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "dashboard")),
		Tab2:    key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "rankings")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c", "ctrl+d"), key.WithHelp("q", "quit")),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Refresh, k.Tab1, k.Tab2, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Refresh, k.Tab1, k.Tab2, k.Quit},
	}
}

func NewApp() appModel {
	h := help.New()
	return appModel{
		activeTab: tabDashboard,
		status:    "  Loading...",
		loading:   true,
		keys:      defaultKeyMap(),
		help:      h,
		aaAPIKey:  os.Getenv("ARTIFICIAL_ANALYSIS_API_KEY"),
		chutesKey: os.Getenv("CHUTES_API_KEY"),
	}
}

func (m appModel) Init() tea.Cmd {
	return tea.Batch(
		m.fetchData(),
		tea.Every(refreshInterval, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}),
	)
}

// Messages
type tickMsg time.Time
type fetchedMsg struct {
	chutes     []model.Chute
	quality    map[string]int
	qualitySrc string
	usage      map[string]any
	usageErr   string
	err        error
}

func (m appModel) fetchData() tea.Cmd {
	return func() tea.Msg {
		var quality map[string]int
		var qualitySrc string
		var err error

		// Fetch quality scores (once per session, but we do it each time for simplicity)
		quality, qualitySrc, err = api.FetchQualityScores(m.aaAPIKey)
		if err == nil && len(quality) > 0 {
			model.SetQualityCache(quality)
		}

		// Fetch chutes
		chutes, fetchErr := api.FetchChutes()
		if fetchErr != nil {
			return fetchedMsg{err: fetchErr}
		}

		// Fetch usage
		usage, usageErrStr, _ := api.FetchUsage(m.chutesKey)

		return fetchedMsg{
			chutes:     chutes,
			quality:    quality,
			qualitySrc: qualitySrc,
			usage:      usage,
			usageErr:   usageErrStr,
			err:        err,
		}
	}
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Refresh):
			m.loading = true
			m.status = "  Fetching data from chutes.ai..."
			return m, m.fetchData()
		case key.Matches(msg, m.keys.Tab1):
			m.activeTab = tabDashboard
			return m, nil
		case key.Matches(msg, m.keys.Tab2):
			m.activeTab = tabRankings
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		return m, nil

	case tickMsg:
		return m, m.fetchData()

	case fetchedMsg:
		m.loading = false
		if msg.err != nil {
			m.status = fmt.Sprintf("  Error: %v", msg.err)
			return m, nil
		}
		m.chutes = msg.chutes
		m.usage = msg.usage
		m.qualitySrc = msg.qualitySrc

		usable := 0
		for _, c := range m.chutes {
			if c.IsUsable() {
				usable++
			}
		}

		now := time.Now().Format("15:04:05")
		usagePart := msg.usageErr
		if usagePart != "" {
			usagePart = "  " + usagePart
		}
		m.status = fmt.Sprintf(
			"  ●  %d chutes  ·  %d usable  ·  %s%s  ·  updated %s  ·  auto-refresh every %ds",
			len(m.chutes), usable, m.qualitySrc, usagePart, now, int(refreshInterval.Seconds()),
		)
		return m, nil
	}

	return m, nil
}

func (m appModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	var content string
	switch m.activeTab {
	case tabDashboard:
		content = ui.DashboardView(m.chutes, m.usage, m.chutesKey != "", m.width)
	case tabRankings:
		content = ui.RankingsView(m.chutes, m.width)
	}

	tabs := m.renderTabs()
	footer := m.renderFooter()

	// Calculate available height for content
	availableHeight := m.height - lipgloss.Height(tabs) - lipgloss.Height(footer) - lipgloss.Height(m.status)
	if availableHeight < 0 {
		availableHeight = 0
	}

	// Truncate content to fit if needed
	lines := strings.Split(content, "\n")
	if len(lines) > availableHeight {
		content = strings.Join(lines[:availableHeight], "\n")
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		tabs,
		content,
		m.status,
		footer,
	)
}

func (m appModel) renderTabs() string {
	dashboardStyle := lipgloss.NewStyle().Padding(0, 2)
	rankingsStyle := lipgloss.NewStyle().Padding(0, 2)

	if m.activeTab == tabDashboard {
		dashboardStyle = dashboardStyle.Background(lipgloss.Color("63")).Foreground(lipgloss.Color("255"))
	} else {
		dashboardStyle = dashboardStyle.Foreground(lipgloss.Color("250"))
	}

	if m.activeTab == tabRankings {
		rankingsStyle = rankingsStyle.Background(lipgloss.Color("63")).Foreground(lipgloss.Color("255"))
	} else {
		rankingsStyle = rankingsStyle.Foreground(lipgloss.Color("250"))
	}

	return lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Render(
			dashboardStyle.Render(" Dashboard ") +
				" " +
				rankingsStyle.Render(" Best Models "),
		)
}

func (m appModel) renderFooter() string {
	return lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Padding(0, 1).
		Render(m.help.View(m.keys))
}


