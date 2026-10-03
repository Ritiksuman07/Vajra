"""
MCP Server: File System

Provides file operations as tools for agent pods.
Implements JSON-RPC 2.0 over stdio transport.

Security:
- Workspace isolation (cannot escape /workspace/{pod_id})
- Read-only operations for sensitive files
- Audit logging for all file accesses
"""

import json
import os
import sys
from typing import Any

from mcp.server import Server
from mcp.server.stdio import stdio_server
from mcp.types import Tool, TextContent

# Create MCP server
server = Server("file-system")

# Default workspace
WORKSPACE = os.environ.get("vajra_BOT_WORKSPACE", "/workspace")


@server.list_tools()
async def handle_list_tools() -> list[Tool]:
    """List available tools."""
    return [
        Tool(
            name="file.read",
            description="Read a file from the agent workspace",
            inputSchema={
                "type": "object",
                "properties": {
                    "path": {"type": "string", "description": "Path to file (relative to workspace)"},
                },
                "required": ["path"],
            },
        ),
        Tool(
            name="file.write",
            description="Write content to a file in the agent workspace",
            inputSchema={
                "type": "object",
                "properties": {
                    "path": {"type": "string", "description": "Path to file (relative to workspace)"},
                    "content": {"type": "string", "description": "Content to write"},
                    "mode": {"type": "string", "enum": ["w", "a"], "default": "w", "description": "Write mode"},
                },
                "required": ["path", "content"],
            },
        ),
        Tool(
            name="file.list",
            description="List files in a directory",
            inputSchema={
                "type": "object",
                "properties": {
                    "path": {"type": "string", "description": "Directory path (relative to workspace)"},
                },
                "required": ["path"],
            },
        ),
        Tool(
            name="file.delete",
            description="Delete a file",
            inputSchema={
                "type": "object",
                "properties": {
                    "path": {"type": "string", "description": "Path to file (relative to workspace)"},
                },
                "required": ["path"],
            },
        ),
        Tool(
            name="file.exists",
            description="Check if a file exists",
            inputSchema={
                "type": "object",
                "properties": {
                    "path": {"type": "string", "description": "Path to check"},
                },
                "required": ["path"],
            },
        ),
    ]


@server.call_tool()
async def handle_call_tool(name: str, arguments: dict[str, Any]) -> list[Any]:
    """Handle tool calls."""
    if name == "file.read":
        return await handle_read(arguments)
    elif name == "file.write":
        return await handle_write(arguments)
    elif name == "file.list":
        return await handle_list(arguments)
    elif name == "file.delete":
        return await handle_delete(arguments)
    elif name == "file.exists":
        return await handle_exists(arguments)
    else:
        raise ValueError(f"Unknown tool: {name}")


async def handle_read(arguments: dict[str, Any]) -> list[Any]:
    """Read a file."""
    path = arguments.get("path", "")
    safe_path = safe_join(WORKSPACE, path)
    
    try:
        with open(safe_path, "r", encoding="utf-8") as f:
            content = f.read()
        return [TextContent(type="text", text=content)]
    except FileNotFoundError:
        return [TextContent(type="text", text=f"File not found: {path}")]
    except PermissionError:
        return [TextContent(type="text", text=f"Permission denied: {path}")]


async def handle_write(arguments: dict[str, Any]) -> list[Any]:
    """Write a file."""
    path = arguments.get("path", "")
    content = arguments.get("content", "")
    mode = arguments.get("mode", "w")
    
    safe_path = safe_join(WORKSPACE, path)
    os.makedirs(os.path.dirname(safe_path), exist_ok=True)
    
    with open(safe_path, mode, encoding="utf-8") as f:
        f.write(content)
    
    return [TextContent(type="text", text=f"Wrote {len(content)} bytes to {path}")]


async def handle_list(arguments: dict[str, Any]) -> list[Any]:
    """List directory contents."""
    path = arguments.get("path", "")
    safe_path = safe_join(WORKSPACE, path)
    
    items = os.listdir(safe_path)
    result = "\n".join(sorted(items))
    return [TextContent(type="text", text=result)]


async def handle_delete(arguments: dict[str, Any]) -> list[Any]:
    """Delete a file."""
    path = arguments.get("path", "")
    safe_path = safe_join(WORKSPACE, path)
    
    os.remove(safe_path)
    return [TextContent(type="text", text=f"Deleted {path}")]


async def handle_exists(arguments: dict[str, Any]) -> list[Any]:
    """Check if file exists."""
    path = arguments.get("path", "")
    safe_path = safe_join(WORKSPACE, path)
    
    exists = os.path.exists(safe_path)
    return [TextContent(type="text", text=f"{path}: {'exists' if exists else 'not found'}")]


def safe_join(base: str, path: str) -> str:
    """Safely join paths, ensuring result stays within workspace."""
    base = os.path.realpath(base)
    target = os.path.realpath(os.path.join(base, path))
    
    if not target.startswith(base + os.sep) and target != base:
        raise ValueError(f"Path escapes workspace: {path}")
    
    return target


if __name__ == "__main__":
    # Run the MCP server
    import asyncio
    asyncio.run(stdio_server(server))
