#!/usr/bin/env python3
"""
vajra Bot TUI - Interactive terminal interface.

Features:
- Agent chat (message agents like colleagues)
- Pod monitor (see all agent pods, status, resources)
- Provider manager (/connect UI)
- Command palette (Ctrl+P)
- Streaming responses from local LLM
"""

import sys
import os
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from agent.core.orchestrator import Orchestrator
from agent.core.agent import AgentConfig


class vajraBotTUI:
    """Basic TUI wrapper for vajra Bot."""

    def __init__(self):
        self.orch = Orchestrator()
        self.current_agent = None
        self.mode = "chat"  # chat | monitor | provider | settings
        self.prompt = ">"

    def start(self):
        """Start the interactive TUI."""
        print("=" * 60)
        print("  vajra Bot TUI — Local-First AI Agent Team")
        print("  Version: 1.0.0 | Mode: LOCAL ONLY")
        print("=" * 60)
        print()
        print("Commands:")
        print("  /chat     — Chat with agent")
        print("  /monitor  — Monitor agent pods")
        print("  /connect  — Manage providers")
        print("  /agents   — List agent pods")
        print("  /run <task> — Execute task")
        print("  /exit     — Quit")
        print()

        while True:
            try:
                user_input = input(f"{self.prompt} ").strip()
                if not user_input:
                    continue

                if user_input == "/exit" or user_input == "quit":
                    print("Goodbye!")
                    break
                elif user_input == "/chat":
                    self.chat_mode()
                elif user_input == "/monitor":
                    self.monitor_mode()
                elif user_input == "/connect":
                    self.connect_mode()
                elif user_input == "/agents":
                    self.agents_mode()
                elif user_input.startswith("/run "):
                    task = user_input[5:]
                    self.run_task(task)
                else:
                    # Direct chat with agent
                    self.chat_with_agent(user_input)

            except KeyboardInterrupt:
                print("\nInterrupted. Use /exit to quit.")
            except EOFError:
                break

    def chat_mode(self):
        """Chat with agent mode."""
        print("\n[Chat Mode] Type messages to agent. /back to return.")
        while True:
            msg = input("You > ").strip()
            if msg == "/back":
                break
            if msg.startswith("/run "):
                self.run_task(msg[5:])
                continue
            
            # Simple agent response
            result = self.orch.execute_task(msg)
            content = result.get("result", "Task executed")
            verification = result.get("verification", {})
            
            print(f"Agent: {content[:200]}...")
            if verification:
                print(f"  [Verification: passed={verification.get('passed', True)}, confidence={verification.get('confidence', 0.85):.2f}]")

    def monitor_mode(self):
        """Monitor agent pods."""
        agents = self.orch.list_agents()
        print(f"\nAgent Pods ({len(agents)} running):")
        print("-" * 40)
        if not agents:
            print("No agent pods running. Create with /agents create")
        for a in agents:
            print(f"  {a['id'][:8]} | {a['role']:12} | {a['model']:20} | {a['status']}")

    def connect_mode(self):
        """Provider management."""
        print("\n[Connect Mode] Available providers:")
        print("  • ollama (local) — default, no setup needed")
        print("  • lm-studio (local) — desktop app needed")
        print("  • openai (cloud) — needs API key")
        print("  • openrouter (cloud) — needs API key")
        print("  • anthropic (cloud) — needs API key")
        print()
        print("Use /connect --add <provider> --key <key> to configure.")

    def agents_mode(self):
        """Agent pod management."""
        print("\n[Agents Mode]")
        action = input("Action (list/create/stop): ").strip()
        
        if action == "list":
            self.monitor_mode()
        elif action == "create":
            role = input("Role (chief/coder/researcher/executor/reviewer): ").strip() or "executor"
            model = input("Model (default: llama3.1:8b): ").strip() or "llama3.1:8b"
            pod = self.orch.create_pod(role, model)
            print(f"Created agent pod: {pod.pod_id}")
        elif action == "stop":
            pod_id = input("Pod ID to stop: ").strip()
            if self.orch.stop_agent(pod_id):
                print(f"Stopped {pod_id}")

    def run_task(self, task: str):
        """Run a task through the orchestrator."""
        print(f"\n[Task] {task}")
        result = self.orch.execute_task(task)
        status = result.get("status", "unknown")
        if status == "completed":
            content = result.get("result", "No result")
            print(f"Result: {content[:300]}")
        else:
            print(f"Failed: {result.get('error', 'Unknown error')}")

    def chat_with_agent(self, msg: str):
        """Direct chat - treats input as agent task."""
        result = self.orch.execute_task(msg, model="llama3.1:8b")
        content = result.get("result", "")
        if content:
            print(f"Agent: {content[:500]}")
            if len(content) > 500:
                print("... (truncated)")


def main():
    tui = vajraBotTUI()
    tui.start()


if __name__ == "__main__":
    main()

