"""
Agent-to-Agent (A2A) Protocol

Defines how agent pods communicate and coordinate work.
Uses JSON-RPC 2.0 over local HTTP/Unix sockets.

Key concepts:
- Task delegation (Chief → Specialist)
- Status updates (Working → Completed/Failed)
- Shared context (group chats, thread history)
- Approval gates (human-in-the-loop for critical actions)
"""

from enum import Enum
from dataclasses import dataclass
from typing import Optional, List, Dict
import time


class TaskStatus(Enum):
    SUBMITTED = "submitted"
    WORKING = "working"
    COMPLETED = "completed"
    FAILED = "failed"
    BLOCKED = "blocked"
    REVIEW_REQUIRED = "review_required"


@dataclass
class A2ATask:
    task_id: str
    sender_id: str
    receiver_id: str
    title: str
    description: str
    status: TaskStatus = TaskStatus.SUBMITTED
    created_at: float = 0.0
    updated_at: float = 0.0
    result: Optional[str] = None
    verification: Optional[dict] = None


class A2AAgent:
    """Agent-to-agent communication layer."""

    def __init__(self, agent_id: str, role: str = "agent"):
        self.agent_id = agent_id
        self.role = role
        self.inbox: List[A2ATask] = []
        self.history: List[dict] = []
        self.peers: Dict[str, dict] = {}

    def send_task(self, task: A2ATask) -> bool:
        """Send a task to another agent."""
        # In production: POST to peer agent's A2A endpoint
        # For now: local delivery
        self.inbox.append(task)
        return True

    def receive_task(self) -> Optional[A2ATask]:
        """Receive the next task from inbox."""
        if self.inbox:
            return self.inbox.pop(0)
        return None

    def update_status(self, task_id: str, status: TaskStatus) -> None:
        """Update task status and notify sender."""
        # Would send status update to sender
        pass

    def broadcast_message(self, message: str, group: Optional[List[str]] = None) -> None:
        """Broadcast a message to group members."""
        # Group chat coordination
        pass


class ChiefOfStaff:
    """Chief of Staff agent for multi-agent coordination."""

    def __init__(self, workspace: str = "/workspace/chief"):
        self.role = "chief_of_staff"
        self.workspace = workspace
        self.sub_agents: Dict[str, A2AAgent] = {}
        self.active_tasks: Dict[str, A2ATask] = {}

    def delegate(self, task: A2ATask, specialist_role: str) -> bool:
        """Delegate task to appropriate specialist agent."""
        if specialist_role in self.sub_agents:
            agent = self.sub_agents[specialist_role]
            agent.send_task(task)
            self.active_tasks[task.task_id] = task
            return True
        return False

    def collect_results(self, task_id: str) -> Optional[str]:
        """Collect results from specialist agents."""
        task = self.active_tasks.get(task_id)
        if task and task.status == TaskStatus.COMPLETED:
            return task.result
        return None

    def coordinate_group_chat(self, members: List[str], message: str) -> None:
        """Coordinate agents in a group chat."""
        for member in members:
            if member in self.sub_agents:
                self.sub_agents[member].broadcast_message(message, group=members)

