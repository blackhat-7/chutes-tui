#!/usr/bin/env python3
"""
chutes_tui — Chutes AI Utilization Dashboard

Two screens:
  1  Dashboard  — live mirror of chutes.ai/app/research/utilization
  2  Rankings   — models ranked by quality × availability (quality-first)

Usage:
    uv run main.py

Keys: r = refresh · 1 = Dashboard · 2 = Rankings · q = quit
"""
from __future__ import annotations

import os
import re

from dotenv import load_dotenv
load_dotenv()
from dataclasses import dataclass
from datetime import datetime

import httpx
from rich.text import Text
from textual import work
from textual.app import App, ComposeResult
from textual.binding import Binding
from textual.containers import Horizontal, Vertical
from textual.widgets import (
    DataTable,
    Footer,
    Header,
    Static,
    TabbedContent,
    TabPane,
)

CHUTES_URL = "https://chutes.ai/app/research/utilization"
CHUTES_API_BASE = "https://api.chutes.ai"
REFRESH_INTERVAL = 60  # seconds

AA_API_KEY     = os.environ.get("ARTIFICIAL_ANALYSIS_API_KEY", "")
CHUTES_API_KEY = os.environ.get("CHUTES_API_KEY", "")

_quality_cache: dict[str, int] = {}
_quality_src: str = ""        # status message set once after first fetch attempt
_my_usage: dict = {}
DEFAULT_QUALITY = 50


# ---------------------------------------------------------------------------
# Data model
# ---------------------------------------------------------------------------

@dataclass
class Chute:
    chute_id: str
    name: str
    utilization_current: float = 0.0
    utilization_5m: float = 0.0
    utilization_1h: float = 0.0
    rate_limit_ratio_5m: float = 0.0
    rate_limit_ratio_1h: float = 0.0
    total_requests_1h: float = 0.0
    completed_requests_1h: float = 0.0
    rate_limited_requests_1h: float = 0.0
    instance_count: int = 0
    active_instance_count: int = 0
    scalable: bool = False
    action_taken: str = ""
    effective_multiplier: float = 0.0
    tee: bool = False

    @property
    def display_name(self) -> str:
        parts = self.name.split("/", 1)
        return parts[1] if len(parts) == 2 else self.name

    @property
    def quality(self) -> int:
        normed = _norm(self.name)
        best, best_len = DEFAULT_QUALITY, 0
        for key, score in _quality_cache.items():
            if key in normed and len(key) > best_len:
                best, best_len = score, len(key)
        return best

    @property
    def availability(self) -> float:
        """0-1: probability of not being rate-limited right now.
        Zero-traffic models are capped at 0.8 — unproven, not guaranteed available."""
        base = max(0.0, 1.0 - self.rate_limit_ratio_1h * 0.65 - self.utilization_1h * 0.35)
        if self.total_requests_1h == 0:
            return min(base, 0.8)
        return base

    @property
    def score(self) -> float:
        """Composite: quality 55%, availability 45%."""
        return self.quality / 100.0 * 0.55 + self.availability * 0.45

    @property
    def is_usable(self) -> bool:
        return self.active_instance_count > 0 and self.rate_limit_ratio_5m < 0.98


# ---------------------------------------------------------------------------
# Parsing — extracts inline JS objects from the SSR'd HTML
# ---------------------------------------------------------------------------

_KV = re.compile(
    r'(\w+):("(?:[^"\\]|\\.)*"|-?\.?\d[\d.]*(?:[eE][+-]?\d+)?|true|false|null)'
)


def _parse_obj(s: str) -> dict:
    out: dict = {}
    for m in _KV.finditer(s):
        k, v = m.group(1), m.group(2)
        if v.startswith('"'):
            out[k] = v[1:-1]
        elif v == "true":
            out[k] = True
        elif v == "false":
            out[k] = False
        elif v == "null":
            out[k] = None
        else:
            try:
                out[k] = float(v)
            except ValueError:
                out[k] = v
    return out


def parse_html(html: str) -> list[Chute]:
    chutes: list[Chute] = []
    seen: set[str] = set()
    for m in re.finditer(r'\{chute_id:[^}]+\}', html):
        p = _parse_obj(m.group())
        cid = p.get("chute_id")
        name = p.get("name", "")
        if not cid or cid in seen or not name or "[private" in str(name).lower():
            continue
        seen.add(str(cid))
        chutes.append(Chute(
            chute_id=str(cid),
            name=str(p.get("name", "")),
            utilization_current=float(p.get("utilization_current") or 0),
            utilization_5m=float(p.get("utilization_5m") or 0),
            utilization_1h=float(p.get("utilization_1h") or 0),
            rate_limit_ratio_5m=float(p.get("rate_limit_ratio_5m") or 0),
            rate_limit_ratio_1h=float(p.get("rate_limit_ratio_1h") or 0),
            total_requests_1h=float(p.get("total_requests_1h") or 0),
            completed_requests_1h=float(p.get("completed_requests_1h") or 0),
            rate_limited_requests_1h=float(p.get("rate_limited_requests_1h") or 0),
            instance_count=int(p.get("instance_count") or 0),
            active_instance_count=int(p.get("active_instance_count") or 0),
            scalable=bool(p.get("scalable")),
            action_taken=str(p.get("action_taken") or ""),
            effective_multiplier=float(p.get("effective_multiplier") or 0),
            tee=bool(p.get("tee")),
        ))
    return chutes


async def fetch_chutes() -> list[Chute]:
    async with httpx.AsyncClient(timeout=30, follow_redirects=True) as client:
        r = await client.get(
            CHUTES_URL,
            headers={"User-Agent": "Mozilla/5.0 (chutes-tui/1.0)"},
        )
        r.raise_for_status()
        return parse_html(r.text)


def _norm(s: str) -> str:
    """Normalize a model name/slug for matching by collapsing all separators."""
    return re.sub(r'[-._/\s]+', '-', s.lower()).strip('-')


async def fetch_quality_scores() -> dict[str, int]:
    """Fetch model quality scores from Artificial Analysis API and normalize to 0-100."""
    if not AA_API_KEY:
        return {}
    async with httpx.AsyncClient(timeout=30) as client:
        r = await client.get(
            "https://artificialanalysis.ai/api/v2/data/llms/models",
            headers={"x-api-key": AA_API_KEY},
        )
        r.raise_for_status()
        models = r.json().get("data", [])

    raw: list[tuple[str, float]] = []
    for m in models:
        score = (m.get("evaluations") or {}).get("artificial_analysis_intelligence_index")
        if score is None:
            continue
        for key in {m.get("slug", ""), m.get("name", "")}:
            if key:
                raw.append((_norm(key), float(score)))

    if not raw:
        return {}

    values = [s for _, s in raw]
    lo, hi = min(values), max(values)
    span = hi - lo or 1
    return {key: int((s - lo) / span * 100) for key, s in raw}


async def fetch_my_usage() -> dict:
    if not CHUTES_API_KEY:
        return {}
    async with httpx.AsyncClient(timeout=15) as client:
        r = await client.get(
            f"{CHUTES_API_BASE}/users/me/subscription_usage",
            headers={"Authorization": f"Bearer {CHUTES_API_KEY}"},
        )
        r.raise_for_status()
        return r.json()


# ---------------------------------------------------------------------------
# Rich text helpers
# ---------------------------------------------------------------------------

def _pct(value: float, *, higher_good: bool = False) -> Text:
    pct = value * 100
    if higher_good:
        col = "bright_green" if pct > 70 else ("yellow" if pct > 40 else "bright_red")
    else:
        col = "bright_red" if pct > 70 else ("yellow" if pct > 30 else "bright_green")
    return Text(f"{pct:5.1f}%", style=col, justify="right")


def _score_bar(score: float, width: int = 10) -> Text:
    filled = round(score * width)
    bar = "█" * filled + "░" * (width - filled)
    col = "bright_green" if score > 0.7 else ("yellow" if score > 0.45 else "bright_red")
    return Text(f"{bar} {score * 100:4.1f}", style=col)


def _qual_cell(q: int) -> Text:
    if q == DEFAULT_QUALITY and _quality_cache:
        return Text("?", style="dim", justify="right")
    col = "bright_green" if q >= 85 else ("yellow" if q >= 72 else "bright_red")
    return Text(str(q), style=col, justify="right")


def _req(n: float) -> Text:
    s = (
        f"{n/1_000_000:.1f}M" if n >= 1_000_000 else
        f"{n/1_000:.1f}K"     if n >= 1_000      else
        f"{n:.0f}"
    )
    return Text(s, justify="right")


# ---------------------------------------------------------------------------
# My usage bar
# ---------------------------------------------------------------------------


class MyUsageBar(Static):
    DEFAULT_CSS = """
    MyUsageBar {
        height: 2;
        padding: 0 2;
        background: $panel;
        border-bottom: tall $primary-darken-3;
        content-align: left middle;
    }
    """

    def refresh_usage(self, usage: dict) -> None:
        if not usage:
            msg = "[dim]  My Usage  ·  loading...[/]" if CHUTES_API_KEY else "[dim]  My Usage  ·  set CHUTES_API_KEY to see personal usage[/]"
            self.update(msg)
            return

        parts: list[str] = ["[bold]My Usage[/]"]
        for key, val in usage.items():
            label = key.replace("_", " ").title()
            if isinstance(val, dict):
                used = val.get("used") or val.get("usage") or 0
                cap  = val.get("cap")  or val.get("limit") or 0
                if cap:
                    ratio = used / cap
                    clamped = min(ratio, 1.0)
                    col = "bright_red" if clamped > 0.85 else ("yellow" if clamped > 0.6 else "bright_green")
                    bar = "█" * round(clamped * 10) + "░" * (10 - round(clamped * 10))
                    pct_str = f"OVER ({ratio*100:.0f}%)" if ratio > 1.0 else f"{ratio*100:.0f}%"
                    parts.append(f"[dim]{label}:[/] [{col}]{bar} {pct_str}[/]")
                elif used:
                    parts.append(f"[dim]{label}:[/] [cyan]{used:,}[/]")
            elif isinstance(val, (int, float)) and val:
                fmt = f"{val:,.2f}" if isinstance(val, float) else f"{val:,}"
                parts.append(f"[dim]{label}:[/] [cyan]{fmt}[/]")

        self.update("  ·  ".join(parts))


# ---------------------------------------------------------------------------
# Stats banner
# ---------------------------------------------------------------------------

class StatsBanner(Horizontal):
    DEFAULT_CSS = """
    StatsBanner {
        height: 5;
        margin: 1 0;
        align: center middle;
    }
    StatsBanner > .stat-card {
        background: $panel;
        border: tall $primary-darken-2;
        padding: 0 2;
        content-align: center middle;
        min-width: 20;
        height: 5;
        margin: 0 1;
    }
    """

    def compose(self) -> ComposeResult:
        for sid in ("active", "instances", "requests", "ratelim", "scalable"):
            yield Static("", id=f"sc-{sid}", classes="stat-card")

    def refresh_stats(self, chutes: list[Chute]) -> None:
        active    = sum(1 for c in chutes if c.active_instance_count > 0)
        instances = sum(c.instance_count for c in chutes)
        reqs      = sum(c.total_requests_1h for c in chutes)
        ratelim   = sum(c.rate_limited_requests_1h for c in chutes)
        scalable  = sum(1 for c in chutes if c.scalable)

        def card(val: str, lbl: str) -> str:
            return f"{val}\n[dim]{lbl}[/]"

        self.query_one("#sc-active",    Static).update(card(f"[bold cyan]{active}[/]",              "Active Models"))
        self.query_one("#sc-instances", Static).update(card(f"[bold cyan]{instances}[/]",           "Instances"))
        self.query_one("#sc-requests",  Static).update(card(f"[bold cyan]{reqs/1000:.1f}K[/]",      "Req / 1h"))
        self.query_one("#sc-ratelim",   Static).update(card(f"[bold yellow]{ratelim/1000:.1f}K[/]", "Rate-Lim / 1h"))
        self.query_one("#sc-scalable",  Static).update(card(f"[bold green]{scalable}[/]",           "Scalable"))


# ---------------------------------------------------------------------------
# Tab 1 — Dashboard (raw utilization mirror)
# ---------------------------------------------------------------------------

class DashboardTab(Vertical):
    DEFAULT_CSS = "DashboardTab { height: 1fr; }"

    def compose(self) -> ComposeResult:
        yield MyUsageBar(id="my-usage")
        yield StatsBanner()
        yield DataTable(id="dash-tbl", cursor_type="row", zebra_stripes=True)

    def on_mount(self) -> None:
        self.query_one("#my-usage", MyUsageBar).refresh_usage({})
        t = self.query_one("#dash-tbl", DataTable)
        t.add_column("Model",     width=38)
        t.add_column("Cur %",     width=8)
        t.add_column("5m %",      width=8)
        t.add_column("1h %",      width=8)
        t.add_column("RL %",      width=8)
        t.add_column("Instances", width=11)
        t.add_column("Req / 1h",  width=11)
        t.add_column("Action",    width=20)

    def populate(self, chutes: list[Chute], usage: dict | None = None) -> None:
        self.query_one(StatsBanner).refresh_stats(chutes)
        if usage is not None:
            self.query_one("#my-usage", MyUsageBar).refresh_usage(usage)
        t = self.query_one("#dash-tbl", DataTable)
        t.clear()

        for c in sorted(chutes, key=lambda c: c.utilization_current, reverse=True):
            name = c.display_name
            name_cell = Text(f"[TEE] {name}" if c.tee else name, style="bold" if c.tee else "")

            ai, ti = c.active_instance_count, c.instance_count
            inst_str = f"{ai}/{ti}"
            inst_cell = (
                Text(inst_str, style="dim")          if ai == 0 else
                Text(inst_str, style="yellow")       if ai < ti else
                Text(inst_str, style="bright_green")
            )

            action = c.action_taken.replace("_", " ").title()
            if "Scale Up" in action:
                act_cell = Text(f"^ {action}", style="yellow")
            elif "Scale Down" in action:
                act_cell = Text(f"v {action}", style="cyan")
            else:
                act_cell = Text(action, style="dim")

            t.add_row(
                name_cell,
                _pct(c.utilization_current),
                _pct(c.utilization_5m),
                _pct(c.utilization_1h),
                _pct(c.rate_limit_ratio_1h),
                inst_cell,
                _req(c.total_requests_1h),
                act_cell,
                key=c.chute_id,
            )


# ---------------------------------------------------------------------------
# Tab 2 — Rankings
# ---------------------------------------------------------------------------

class RankingsTab(Vertical):
    DEFAULT_CSS = """
    RankingsTab { height: 1fr; }
    .rank-hint {
        height: 2;
        padding: 0 2;
        background: $panel;
        border-bottom: tall $primary-darken-3;
    }
    """

    def compose(self) -> ComposeResult:
        yield Static(
            "[dim]Score = Quality 55% + Availability 45%  "
            "(Availability = 1 - rate-limit% - util%)  "
            "Models with active instances  ·  [bright_red]RL[/][dim] = heavily rate-limited[/]",
            classes="rank-hint",
        )
        yield DataTable(id="rank-tbl", cursor_type="row", zebra_stripes=True)

    def on_mount(self) -> None:
        t = self.query_one("#rank-tbl", DataTable)
        t.add_column("#",       width=4)
        t.add_column("Model",   width=38)
        t.add_column("Qual",    width=6)
        t.add_column("Avail%",  width=8)
        t.add_column("RL %",    width=8)
        t.add_column("Util %",  width=8)
        t.add_column("Score",   width=18)
        t.add_column("Inst",    width=6)
        t.add_column("Flags",   width=8)

    def populate(self, chutes: list[Chute]) -> None:
        t = self.query_one("#rank-tbl", DataTable)
        t.clear()

        ranked = sorted(
            (c for c in chutes if c.active_instance_count > 0),
            key=lambda c: c.score,
            reverse=True,
        )

        for rank, c in enumerate(ranked, 1):
            rl = c.rate_limit_ratio_5m >= 0.98
            if rank == 1 and not rl:
                rank_sty, name_sty = "bold bright_yellow", "bold bright_green"
            elif rank <= 3 and not rl:
                rank_sty, name_sty = "bold bright_green", "bold"
            elif rank <= 10 and not rl:
                rank_sty, name_sty = "bold yellow", ""
            elif rl:
                rank_sty, name_sty = "dim", "dim"
            else:
                rank_sty, name_sty = "dim", ""

            flags = Text()
            if c.tee:
                flags.append("TEE", style="bold bright_cyan")
            if rl:
                flags.append(" RL" if c.tee else "RL", style="bold bright_red")

            t.add_row(
                Text(str(rank),             style=rank_sty, justify="right"),
                Text(c.display_name,        style=name_sty),
                _qual_cell(c.quality),
                _pct(c.availability,        higher_good=True),
                _pct(c.rate_limit_ratio_1h),
                _pct(c.utilization_1h),
                _score_bar(c.score),
                Text(str(c.active_instance_count), justify="right"),
                flags,
                key=c.chute_id,
            )


# ---------------------------------------------------------------------------
# App
# ---------------------------------------------------------------------------

APP_CSS = """
Screen { background: $background; }
Header { background: $primary-darken-3; }
#status-bar {
    height: 1;
    background: $panel;
    padding: 0 2;
    dock: bottom;
}
TabbedContent ContentSwitcher { height: 1fr; }
"""


class ChutesApp(App):
    TITLE = "Chutes AI"
    SUB_TITLE = "chutes.ai/app/research/utilization"
    CSS = APP_CSS
    BINDINGS = [
        Binding("r",      "refresh",               "Refresh",   show=True),
        Binding("1",      "switch_tab('dashboard')", "Dashboard", show=True),
        Binding("2",      "switch_tab('rankings')",  "Rankings",  show=True),
        Binding("q",      "quit",                  "Quit",      show=True),
        Binding("ctrl+c", "quit",                  "Quit",      show=False),
        Binding("ctrl+d", "quit",                  "Quit",      show=False),
    ]

    _chutes: list[Chute] = []
    _usage: dict = {}

    def compose(self) -> ComposeResult:
        yield Header(show_clock=True)
        with TabbedContent(initial="dashboard"):
            with TabPane("Dashboard", id="dashboard"):
                yield DashboardTab(id="tab-dash")
            with TabPane("Best Models", id="rankings"):
                yield RankingsTab(id="tab-rank")
        yield Static("  Loading...", id="status-bar")
        yield Footer()

    def on_mount(self) -> None:
        self._load()
        self.set_interval(REFRESH_INTERVAL, self._load)

    @work(exclusive=True, group="fetch")
    async def _load(self) -> None:
        global _quality_cache, _quality_src
        self._set_status("  Fetching data from chutes.ai...")
        if not _quality_src:  # attempt once per session
            if AA_API_KEY:
                try:
                    _quality_cache = await fetch_quality_scores()
                    _quality_src = f"[cyan]live scores ({len(_quality_cache)} models)[/]"
                except httpx.HTTPStatusError as exc:
                    _quality_src = f"[yellow]quality API {exc.response.status_code} — using fallback[/]"
                except httpx.RequestError as exc:
                    _quality_src = f"[yellow]quality fetch failed ({exc.__class__.__name__}) — using fallback[/]"
            else:
                _quality_src = "[dim]fallback scores (set ARTIFICIAL_ANALYSIS_API_KEY for live)[/]"

        data = await fetch_chutes()
        self._chutes = data

        usage_note = ""
        if CHUTES_API_KEY:
            try:
                self._usage = await fetch_my_usage()
            except httpx.HTTPStatusError as exc:
                usage_note = f"  [yellow]usage API {exc.response.status_code}[/]"
            except httpx.RequestError as exc:
                usage_note = f"  [yellow]usage fetch failed: {exc}[/]"

        now = datetime.now().strftime("%H:%M:%S")
        usable = sum(1 for c in data if c.is_usable)
        self._set_status(
            f"  [green]●[/]  {len(data)} chutes  ·  "
            f"[green]{usable} usable[/]  ·  "
            f"{_quality_src}{usage_note}  ·  "
            f"[dim]updated {now}  ·  auto-refresh every {REFRESH_INTERVAL}s[/]"
        )
        self._populate()

    def _populate(self) -> None:
        try:
            self.query_one("#tab-dash", DashboardTab).populate(self._chutes, self._usage)
        except Exception:
            pass
        try:
            self.query_one("#tab-rank", RankingsTab).populate(self._chutes)
        except Exception:
            pass

    def _set_status(self, msg: str) -> None:
        try:
            self.query_one("#status-bar", Static).update(msg)
        except Exception:
            pass

    def action_refresh(self) -> None:
        self._load()

    def action_switch_tab(self, tab_id: str) -> None:
        try:
            self.query_one(TabbedContent).active = tab_id
        except Exception:
            pass


if __name__ == "__main__":
    ChutesApp().run()
