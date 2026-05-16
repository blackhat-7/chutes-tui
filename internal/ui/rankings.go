package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/blackhat-7/chutes-tui/internal/model"
)

func RankingsView(chutes []model.Chute, width int) string {
	var b strings.Builder

	b.WriteString(DimStyle.Render("Score = Quality 55% + Availability 45%  (Availability = 1 - rate-limit% - util%)  Models with active instances  ·  "))
	b.WriteString(DangerStyle.Render("RL"))
	b.WriteString(DimStyle.Render(" = heavily rate-limited"))
	b.WriteString("\n\n")

	var ranked []model.Chute
	for _, c := range chutes {
		if c.ActiveInstanceCount > 0 {
			ranked = append(ranked, c)
		}
	}
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].Score() > ranked[j].Score()
	})

	rows := make([][]string, 0, len(ranked))
	hasCache := model.HasQualityCache()

	for rank, c := range ranked {
		rl := c.RateLimitRatio5m >= 0.98
		rankNum := rank + 1

		nameStyle := RankStyle(rankNum, rl)
		name := nameStyle.Render(c.DisplayName())

		var flags strings.Builder
		if c.TEE {
			flags.WriteString(BrightCyanStyle.Render("TEE"))
		}
		if rl {
			if c.TEE {
				flags.WriteString(" ")
			}
			flags.WriteString(DangerStyle.Render("RL"))
		}

		rows = append(rows, []string{
			fmt.Sprintf("%d", rankNum),
			name,
			QualCell(c.Quality(), hasCache),
			Pct(c.Availability(), true),
			Pct(c.RateLimitRatio1h, false),
			Pct(c.Utilization1h, false),
			ScoreBar(c.Score(), 10),
			fmt.Sprintf("%d", c.ActiveInstanceCount),
			flags.String(),
		})
	}

	headers := []string{"#", "Model", "Qual", "Avail%", "RL %", "Util %", "Score", "Inst", "Flags"}
	colWidths := []int{4, 38, 6, 8, 8, 8, 18, 6, 8}

	b.WriteString(renderTable(headers, rows, colWidths, width))
	return b.String()
}
