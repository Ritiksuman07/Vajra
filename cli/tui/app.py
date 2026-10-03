"""
vajra Bot TUI — Full interactive terminal interface.

Built with Bubble Tea (Go) / Textual (Python) principles.
Lightweight fallback using plain Python for compatibility.
"""

from typing import Optional


class TUIApp:
    """vajra Bot interactive TUI application."""

    def __init__(self, standalone: bool = False):
        self.standalone = standalone
        self.theme = "dark"
        self.panels = {
            "chat": {"visible": True, "focus": True},
            "monitor": {"visible": False, "focus": False},
            "provider": {"visible": False, "focus": False},
        }

    def render_chat(self, messages: list[dict]) -> str:
        """Render chat panel."""
        lines = ["─" * 60, " CHAT"]
        for msg in messages[-20:]:
            role = msg.get("role", "user")
            content = str(msg.get("content", ""))[:50]
            lines.append(f" {role.upper():6}: {content}")
        return "\n".join(lines)

    def render_monitor(self, pods: list[dict]) -> str:
        """Render pod monitor panel."""
        lines = ["─" * 60, " MONITOR"]
        lines.append(f"{'ID':<10} {'Role':<12} {'Model':<16} {'Status'}")
        lines.append("─" * 60)
        for pod in pods:
            lines.append(
                f"{pod['id'][:8]:<10} {pod['role']:<12} {pod['model']:<16} {pod['status']}"
            )
        return "\n".join(lines)

    def render_provider(self, providers: list[dict]) -> str:
        """Render provider manager panel."""
        lines = ["─" * 60, " PROVIDERS (/connect)"]
        for p in providers:
            status = "✓" if p.get("status") == "available" else "⚠"
            lines.append(f" {status} {p['name']:<12} {p['type']:<10} {p.get('auth', 'none')}")
        return "\n".join(lines)

    def switch_panel(self, panel: str) -> None:
        """Switch active panel."""
        for name in self.panels:
            self.panels[name]["visible"] = (name == panel)
            self.panels[name]["focus"] = (name == panel)

    def handle_keypress(self, key: str) -> Optional[str]:
        """Handle key bindings."""
        bindings = {
            "ctrl+p": lambda: "command_palette",
            "ctrl+n": lambda: "new_chat",
            "ctrl+m": lambda: "monitor",
            "ctrl+q": lambda: "quit",
        }
        return bindings.get(key, lambda: None)()
