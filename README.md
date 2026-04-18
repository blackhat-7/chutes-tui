# chutes-tui

Terminal dashboard for [chutes.ai](https://chutes.ai) — monitor model utilization and find the best available models in real time.

![demo](demo.gif)

## Features

- **Dashboard** — live mirror of chutes.ai/app/research/utilization with current utilization, rate-limit ratios, and instance counts
- **Best Models** — models ranked by a composite score (quality 55% + availability 45%), with rate-limited models flagged rather than hidden
- **Personal usage** — shows your subscription usage percentage if `CHUTES_API_KEY` is set
- Auto-refreshes every 60 seconds

## Setup

```bash
uv run main.py
```

**Optional env vars** (create a `.env` file):

```
CHUTES_API_KEY=...              # personal usage bar
ARTIFICIAL_ANALYSIS_API_KEY=... # live quality scores (from artificialanalysis.ai)
```

## Keys

| Key | Action |
|-----|--------|
| `1` | Dashboard |
| `2` | Best Models |
| `r` | Refresh |
| `q` / `ctrl+c` / `ctrl+d` | Quit |
