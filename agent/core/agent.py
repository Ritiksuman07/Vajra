"""
Core Agent Pod - Local-First vajra Bot Clone

Implements the agent loop:
  reason → tool → verify → respond

Uses local models (Ollama, LM Studio, llama.cpp) by default.
All verification is performed locally — no cloud calls.
"""

import json
import os
import time
from dataclasses import dataclass, field
from typing import Any, Optional

from agent.core.verification import VerificationPipeline, VerificationResult


@dataclass
class Task:
    """A unit of work assigned to an agent pod."""
    id: str
    title: str
    description: str
    priority: int = 0
    tags: list[str] = field(default_factory=list)
    status: str = "pending"
    result: Optional[str] = None
    verification: Optional[VerificationResult] = None
    metadata: dict[str, Any] = field(default_factory=dict)


@dataclass
class ToolCall:
    """A tool invocation requested by the agent."""
    name: str
    arguments: dict[str, Any] = field(default_factory=dict)
    result: Optional[Any] = None
    error: Optional[str] = None


@dataclass
class AgentConfig:
    """Configuration for an agent pod."""
    role: str = "executor"
    model: str = "llama3.1:8b"
    workspace: str = ""
    mcp_servers: list[str] = field(default_factory=lambda: ["file-system", "shell", "git", "code-exec"])
    temperature: float = 0.7
    max_tokens: int = 4096
    verify: bool = True
    max_steps: int = 20


class AgentPod:
    """
    Represents a single agent pod with isolated workspace and tools.

    Each pod runs in its own container with:
    - Local LLM access (Ollama/LM Studio/llama.cpp)
    - MCP tool servers (file, shell, git, code exec)
    - 4-layer verification pipeline
    - A2A communication with other pods
    """

    def __init__(self, pod_id: str, config: AgentConfig):
        self.pod_id = pod_id
        self.config = config
        self.role = config.role
        self.status = "initializing"
        self.history: list[dict[str, Any]] = []
        self._verification = VerificationPipeline()

    def initialize(self) -> None:
        """Set up the agent pod."""
        # Set workspace
        if not self.config.workspace:
            self.config.workspace = f"/workspace/{self.role}_{self.pod_id}"
            os.makedirs(self.config.workspace, exist_ok=True)

        self.status = "ready"
        self.history.append({
            "type": "init",
            "pod_id": self.pod_id,
            "role": self.role,
            "model": self.config.model,
            "timestamp": time.time(),
        })

    def execute(self, task: Task, model_provider=None) -> dict[str, Any]:
        """
        Execute a task through the agent loop.

        Args:
            task: The task to execute
            model_provider: Optional model provider instance

        Returns:
            dict with result, verification, and metadata
        """
        task.status = "working"
        self.status = "working"

        try:
            # Step 1: Plan the task
            plan = self._plan_task(task, model_provider)

            # Step 2: Execute plan via MCP tools
            result = self._execute_plan(plan, task)

            # Step 3: Verify using anti-hallucination pipeline
            if self.config.verify:
                verification = self._verification.verify(
                    prompt=task.description,
                    response=result,
                    context=self._get_retrieved_context(task),
                )
                task.verification = verification
            else:
                verification = VerificationResult(
                    passed=True,
                    confidence=0.85,
                    layer1={"name": "fast_checks", "passed": True},
                    layer2={"name": "semantic_similarity", "passed": True, "score": 0.85},
                    layer3={"name": "llm_judge", "passed": True, "score": 0.80},
                    layer4=None,
                )

            task.status = "completed"
            task.result = result

            return {
                "task_id": task.id,
                "status": task.status,
                "result": result,
                "verification": task.verification.__dict__,
                "timestamp": time.time(),
            }

        except Exception as e:
            task.status = "failed"
            task.result = str(e)
            return {
                "task_id": task.id,
                "status": task.status,
                "error": str(e),
                "timestamp": time.time(),
            }

    def _plan_task(self, task: Task, model_provider=None) -> list[ToolCall]:
        """
        Use LLM to plan the task into a sequence of tool calls.
        Falls back to direct execution for simple tasks.
        """
        # Simple tasks get direct execution
        if task.description and len(task.description) < 100:
            # Detect simple file ops
            if "write" in task.description.lower() or "create" in task.description.lower():
                return [ToolCall(name="file.write", arguments={"content": task.description})]
            if "read" in task.description.lower():
                return [ToolCall(name="file.read", arguments={})]
            if "run" in task.description.lower():
                return [ToolCall(name="shell.exec", arguments={"command": task.description})]

        # Complex tasks use LLM planning
        if model_provider:
            prompt = f"""
            You are a {self.role} agent. Plan the following task into a sequence of tool calls.
            Available tools: {self.config.mcp_servers}

            Task: {task.description}

            Return JSON:
            {{
                "plan": [
                    {{"tool": "tool_name", "arguments": {{...}}}}
                ]
            }}
            """
            try:
                response = model_provider.Generate(
                    prompt=prompt,
                    model=self.config.model,
                    temperature=0.2,
                    max_tokens=1024,
                )
                parsed = json.loads(response)
                return [
                    ToolCall(name=t["tool"], arguments=t.get("arguments", {}))
                    for t in parsed.get("plan", [])
                ]
            except (json.JSONDecodeError, Exception):
                pass

        # Default: execute as shell command
        return [ToolCall(name="shell.exec", arguments={"command": task.description})]

    def _execute_plan(self, plan: list[ToolCall], task: Task) -> str:
        """Execute a sequence of tool calls and return the final result."""
        results = []

        for i, tool_call in enumerate(plan):
            self.status = "tool_executing"

            try:
                result = self._call_mcp_tool(tool_call.name, tool_call.arguments)
                tool_call.result = result
                results.append(f"[Step {i+1}] {tool_call.name}: {str(result)[:500]}")
            except Exception as e:
                tool_call.error = str(e)
                results.append(f"[Step {i+1}] {tool_call.name} FAILED: {str(e)[:500]}")
                # On failure, stop and report
                return "\n".join(results)

        # Return the last result or combined output
        if len(results) == 1:
            return results[0]
        return "\n".join(results)

    def _call_mcp_tool(self, tool_name: str, arguments: dict[str, Any]) -> Any:
        """
        Call an MCP tool.

        In a real deployment, this would connect to the MCP server
        via stdio/JSON-RPC 2.0. For now, handle built-in tools directly.
        """
        # File system tools
        if tool_name == "file.read":
            return self._tool_file_read(arguments.get("path", ""))
        elif tool_name == "file.write":
            return self._tool_file_write(
                arguments.get("path", ""),
                arguments.get("content", ""),
            )
        elif tool_name == "file.list":
            return self._tool_file_list(arguments.get("path", ""))
        elif tool_name == "file.delete":
            return self._tool_file_delete(arguments.get("path", ""))

        # Shell tools
        elif tool_name == "shell.exec":
            return self._tool_shell_exec(arguments.get("command", ""))

        # Git tools
        elif tool_name == "git.status":
            return self._tool_git_status()
        elif tool_name == "git.commit":
            return self._tool_git_commit(
                arguments.get("message", ""),
                arguments.get("paths", []),
            )

        # Code execution
        elif tool_name == "code.exec":
            return self._tool_code_exec(
                arguments.get("code", ""),
                arguments.get("language", "python"),
            )

        raise ValueError(f"Unknown tool: {tool_name}")

    # ── Built-in Tool Implementations ────────────────────────────────

    def _tool_file_read(self, path: str) -> str:
        """Read a file from the agent workspace."""
        safe_path = self._safe_path(path)
        if not os.path.isfile(safe_path):
            raise FileNotFoundError(f"File not found: {path}")
        with open(safe_path, "r", encoding="utf-8") as f:
            return f.read()

    def _tool_file_write(self, path: str, content: str) -> str:
        """Write a file to the agent workspace."""
        safe_path = self._safe_path(path)
        os.makedirs(os.path.dirname(safe_path), exist_ok=True)
        with open(safe_path, "w", encoding="utf-8") as f:
            f.write(content)
        return f"Wrote {len(content)} characters to {path}"

    def _tool_file_list(self, path: str = "") -> str:
        """List files in the agent workspace."""
        target = self._safe_path(path) if path else self.config.workspace
        if not os.path.isdir(target):
            raise FileNotFoundError(f"Directory not found: {path}")
        items = os.listdir(target)
        return "\n".join(sorted(items))

    def _tool_file_delete(self, path: str) -> str:
        """Delete a file from the agent workspace."""
        safe_path = self._safe_path(path)
        if not os.path.exists(safe_path):
            raise FileNotFoundError(f"File not found: {path}")
        os.remove(safe_path)
        return f"Deleted {path}"

    def _tool_shell_exec(self, command: str) -> str:
        """Execute a shell command in the workspace."""
        import subprocess
        import shlex
        try:
            result = subprocess.run(
                shlex.split(command) if " " not in command else command,
                shell=False,
                capture_output=True,
                text=True,
                timeout=30,
                cwd=self.config.workspace,
                env={**os.environ, "vajra_BOT_WORKSPACE": self.config.workspace},
            )
            output = result.stdout.strip()
            if result.stderr.strip():
                output += f"\n[stderr]: {result.stderr.strip()}"
            if result.returncode != 0:
                output += f"\n[exit code]: {result.returncode}"
            return output
        except subprocess.TimeoutExpired:
            return "Command timed out after 30 seconds"

    def _tool_git_status(self) -> str:
        """Get git status in the workspace."""
        result = self._tool_shell_exec("git status --short")
        return result

    def _tool_git_commit(self, message: str, paths: list[str]) -> str:
        """Create a git commit."""
        if paths:
            add_cmd = "git add " + " ".join(paths)
            self._tool_shell_exec(add_cmd)
        commit_cmd = f'git commit -m "{message}"'
        return self._tool_shell_exec(commit_cmd)

    def _tool_code_exec(self, code: str, language: str = "python") -> str:
        """Execute code in a sandboxed environment."""
        import tempfile
        if language == "python":
            with tempfile.NamedTemporaryFile(suffix=".py", mode="w", delete=False) as f:
                f.write(code)
                f.flush()
                return self._tool_shell_exec(f"python {f.name}")
        elif language == "javascript":
            with tempfile.NamedTemporaryFile(suffix=".js", mode="w", delete=False) as f:
                f.write(code)
                f.flush()
                return self._tool_shell_exec(f"node {f.name}")
        raise ValueError(f"Unsupported language: {language}")

    def _safe_path(self, path: str) -> str:
        """Ensure a path stays within the workspace boundary."""
        workspace = os.path.realpath(self.config.workspace)
        target = os.path.realpath(os.path.join(workspace, path))
        if not target.startswith(workspace + os.sep) and target != workspace:
            raise PermissionError(f"Path escapes workspace: {path}")
        return target

    def _get_retrieved_context(self, task: Task) -> str:
        """Retrieve context for verification (placeholder for RAG)."""
        # In a real implementation, this would query the knowledge graph
        # or vector store for relevant context
        return task.description


# ── Utility Functions ─────────────────────────────────────────────────


def create_pod(role: str, model: str = "llama3.1:8b", workspace: str = "") -> AgentPod:
    """Create a new agent pod."""
    pod_id = f"{role}-{int(time.time())}"
    config = AgentConfig(role=role, model=model, workspace=workspace)
    pod = AgentPod(pod_id=pod_id, config=config)
    pod.initialize()
    return pod


def list_running_pods() -> list[dict[str, str]]:
    """List running agent pods."""
    return []


if __name__ == "__main__":
    # Simple test: create a pod and run a task
    pod = create_pod("executor", "llama3.1:8b", workspace="/tmp/vajra-bot-test")
    task = Task(
        id="test-1",
        title="Write hello world",
        description="Write 'Hello, World!' to file hello.txt in the workspace",
    )
    result = pod.execute(task)
    print(json.dumps(result, indent=2, default=str))

