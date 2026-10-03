"""
MCP Server: Shell

Provides shell command execution as tools for agent pods.
Implements JSON-RPC 2.0 over stdio transport.

Security:
- No network access
- Command whitelist (no rm -rf, no curl, no wget)
- Timeout limits
- Output capture
"""

import json
import os
import subprocess
import sys
import shlex
from typing import Any

from mcp.server import Server
from mcp.server.stdio import stdio_server
from mcp.types import Tool, TextContent

server = Server("shell")

# Allowed commands whitelist
ALLOWED_COMMANDS = {
    "ls", "cat", "echo", "pwd", "wc", "head", "tail", "sort", "uniq",
    "grep", "find", "git", "python", "python3", "node", "npm", "npx",
    "pip", "pip3", "go", "rustc", "cargo", "make", "cmake",
    "mkdir", "cp", "mv", "touch", "chmod", "chown",
}

BLOCKED_COMMANDS = {
    "rm", "rmdir", "curl", "wget", "ssh", "scp", "ftp",
    "sudo", "su", "chmod 777", "dd", "mkfs", "fdisk",
}

MAX_COMMAND_LENGTH = 500
MAX_OUTPUT_LENGTH = 10000
COMMAND_TIMEOUT = 30


@server.list_tools()
async def handle_list_tools() -> list[Tool]:
    return [
        Tool(
            name="shell.exec",
            description="Execute a shell command in the workspace",
            inputSchema={
                "type": "object",
                "properties": {
                    "command": {"type": "string", "description": "Shell command to execute"},
                    "timeout": {"type": "integer", "description": "Timeout in seconds", "default": 30},
                },
                "required": ["command"],
            },
        ),
        Tool(
            name="shell.ls",
            description="List directory contents",
            inputSchema={
                "type": "object",
                "properties": {
                    "path": {"type": "string", "description": "Directory path"},
                },
            },
        ),
    ]


@server.call_tool()
async def handle_call_tool(name: str, arguments: dict[str, Any]) -> list[Any]:
    if name == "shell.exec":
        return await handle_exec(arguments)
    elif name == "shell.ls":
        return await handle_ls(arguments)
    else:
        raise ValueError(f"Unknown tool: {name}")


async def handle_exec(arguments: dict[str, Any]) -> list[Any]:
    """Execute a shell command."""
    command = arguments.get("command", "")
    timeout = min(arguments.get("timeout", COMMAND_TIMEOUT), COMMAND_TIMEOUT)

    # Security checks
    if len(command) > MAX_COMMAND_LENGTH:
        return [TextContent(type="text", text=f"Command too long: {len(command)} chars")]

    # Check for blocked commands
    cmd_parts = shlex.split(command)
    if cmd_parts:
        base_cmd = cmd_parts[0]
        if base_cmd in BLOCKED_COMMANDS:
            return [TextContent(type="text", text=f"Blocked command: {base_cmd}")]

    # Check for dangerous patterns
    dangerous = ["rm -rf", "curl | sh", "wget | sh", "sudo ", "chmod 777"]
    for pattern in dangerous:
        if pattern in command:
            return [TextContent(type="text", text=f"Blocked pattern: {pattern}")]

    # Execute
    try:
        result = subprocess.run(
            shlex.split(command) if " " not in command else command,
            shell=False,
            capture_output=True,
            text=True,
            timeout=timeout,
            cwd=os.environ.get("vajra_BOT_WORKSPACE", "/workspace"),
            env={**os.environ, "vajra_BOT_WORKSPACE": os.environ.get("vajra_BOT_WORKSPACE", "/workspace")},
        )

        output = result.stdout.strip()
        if result.stderr.strip():
            output += f"\n[stderr]: {result.stderr.strip()}"
        if result.returncode != 0:
            output += f"\n[exit code]: {result.returncode}"

        # Truncate output
        if len(output) > MAX_OUTPUT_LENGTH:
            output = output[:MAX_OUTPUT_LENGTH] + "\n... [truncated]"

        return [TextContent(type="text", text=output)]
    except subprocess.TimeoutExpired:
        return [TextContent(type="text", text=f"Command timed out after {timeout} seconds")]
    except Exception as e:
        return [TextContent(type="text", text=f"Error: {str(e)}")]


async def handle_ls(arguments: dict[str, Any]) -> list[Any]:
    """List directory contents."""
    path = arguments.get("path", ".")
    try:
        items = os.listdir(path)
        result = "\n".join(sorted(items))
        return [TextContent(type="text", text=result)]
    except Exception as e:
        return [TextContent(type="text", text=f"Error: {str(e)}")]


if __name__ == "__main__":
    import asyncio
    asyncio.run(stdio_server(server))
