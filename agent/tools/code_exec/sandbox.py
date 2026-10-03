"""
MCP Server: Code Execution Sandbox

Provides sandboxed code execution as tools for agent pods.
Security isolation via subprocess + timeouts.

Supported languages:
- Python (default)
- JavaScript (Node.js)
- TypeScript (ts-node or npx)
"""

import json
import os
import subprocess
import tempfile
import uuid
from typing import Any

# Sandbox configuration
SANDBOX_DIR = os.environ.get("vajra_BOT_SANDBOX_DIR", "/tmp/vajra-bot-sandbox")
MAX_EXECUTION_TIME = 30
MAX_OUTPUT_SIZE = 10000


class CodeSandbox:
    """Sandboxed code execution environment."""

    def __init__(self, sandbox_dir: str = SANDBOX_DIR):
        self.sandbox_dir = sandbox_dir
        os.makedirs(sandbox_dir, exist_ok=True)

    def run(self, code: str, language: str = "python", timeout: int = MAX_EXECUTION_TIME) -> str:
        """Execute code in sandboxed environment."""
        lang_config = self._get_lang_config(language)
        if not lang_config:
            return f"Error: Unsupported language: {language}"

        # Create unique temp file
        file_id = str(uuid.uuid4())
        file_path = os.path.join(self.sandbox_dir, f"{file_id}{lang_config['ext']}")

        # Write code
        with open(file_path, "w") as f:
            f.write(code)

        try:
            # Execute in sandbox
            cmd = lang_config["run_cmd"](file_path)
            result = subprocess.run(
                cmd,
                shell=False,
                capture_output=True,
                text=True,
                timeout=timeout,
                cwd=sandbox_dir,
                env={
                    **os.environ,
                    "vajra_BOT_SANDBOX_DIR": sandbox_dir,
                    "PYTHONPATH": sandbox_dir,
                },
            )

            output = result.stdout.strip()
            if result.stderr.strip():
                output += f"\n[stderr]: {result.stderr.strip()}"
            if result.returncode != 0:
                output += f"\n[exit code]: {result.returncode}"

            return self._truncate_output(output)

        except subprocess.TimeoutExpired:
            return f"Code execution timed out after {timeout} seconds"
        except Exception as e:
            return f"Execution error: {str(e)}"
        finally:
            # Cleanup
            try:
                os.remove(file_path)
            except:
                pass

    def _get_lang_config(self, language: str) -> dict:
        """Get language-specific configuration."""
        configs = {
            "python": {
                "ext": ".py",
                "run_cmd": lambda f: ["python", f],
            },
            "javascript": {
                "ext": ".js",
                "run_cmd": lambda f: ["node", f],
            },
            "typescript": {
                "ext": ".ts",
                "run_cmd": lambda f: ["npx", "ts-node", f],
            },
            "bash": {
                "ext": ".sh",
                "run_cmd": lambda f: ["bash", f],
            },
        }
        return configs.get(language)

    def _truncate_output(self, output: str) -> str:
        """Truncate output to max size."""
        if len(output) > MAX_OUTPUT_SIZE:
            return output[:MAX_OUTPUT_SIZE] + "\n... [truncated]"
        return output


# MCP interface
SANDBOX = CodeSandbox()


def execute_code(code: str, language: str = "python", timeout: int = MAX_EXECUTION_TIME) -> str:
    """Execute code in sandbox (MCP tool interface)."""
    return SANDBOX.run(code, language, timeout)


if __name__ == "__main__":
    # Test
    code = "print('Hello from vajra Bot sandbox')"
    result = execute_code(code, "python")
    print(result)

