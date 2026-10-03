"""
Multi-Agent Orchestrator

Coordinates multiple agent pods for complex tasks.
Routes work to appropriate specialists based on task type.

Specialist Roles:
- chief_of_staff: Delegates tasks, coordinates team
- coder: Code-related tasks (write, read, test, refactor)
- researcher: Research and synthesis tasks
- executor: Shell commands, file operations
- reviewer: Code review, security audit
"""

from typing import Optional
import uuid
import time

from agent.core.agent import AgentPod, AgentConfig, Task
from agent.core.a2a_protocol import ChiefOfStaff, A2ATask, TaskStatus, A2AAgent


class Orchestrator:
    """Manages multiple agent pods and their tasks."""

    def __init__(self):
        self.chief = ChiefOfStaff()
        self.pods: dict[str, AgentPod] = {}
        self.running_tasks: dict[str, Task] = {}

    def create_pod(self, role: str, model: str = "llama3.1:8b") -> AgentPod:
        """Create and register a new agent pod."""
        pod_id = f"{role}-{int(time.time())}"
        config = AgentConfig(role=role, model=model)
        pod = AgentPod(pod_id=pod_id, config=config)
        pod.initialize()

        self.pods[pod_id] = pod

        # Register as A2A peer
        a2a_agent = A2AAgent(agent_id=pod_id, role=role)
        self.chief.sub_agents[pod_id] = a2a_agent

        return pod

    def execute_task(self, task_description: str, model: str = "llama3.1:8b") -> dict:
        """Execute a task using the appropriate agent."""
        task_id = str(uuid.uuid4())
        task = Task(
            id=task_id,
            title=f"Task-{task_id[:8]}",
            description=task_description,
        )
        self.running_tasks[task_id] = task

        # Determine best agent by task type
        agent = self._route_task(task_description)

        # Execute the task
        result = agent.execute(task, model_provider=None)
        del self.running_tasks[task_id]
        return result

    def _route_task(self, task_description: str) -> AgentPod:
        """Route task to appropriate agent based on content."""
        desc_lower = task_description.lower()

        # Code-related tasks
        if any(kw in desc_lower for kw in ["code", "python", "function", "class", "test", "bug", "refactor"]):
            role = "coder"
        # Research tasks
        elif any(kw in desc_lower for kw in ["search", "research", "find", "analyze", "summarize"]):
            role = "researcher"
        # Execution tasks
        elif any(kw in desc_lower for kw in ["run", "execute", "create", "write", "delete", "list"]):
            role = "executor"
        # Default to executor
        else:
            role = "executor"

        # Find or create agent of appropriate role
        pod_id = f"{role}-{int(time.time())}"
        if pod_id in self.pods:
            return self.pods[pod_id]

        # Create new pod
        return self.create_pod(role)

    def list_agents(self) -> list[dict]:
        """List all running agent pods."""
        return [
            {
                "id": pod_id,
                "role": pod.config.role,
                "model": pod.config.model,
                "status": pod.status,
            }
            for pod_id, pod in self.pods.items()
        ]

    def stop_agent(self, pod_id: str) -> bool:
        """Stop an agent pod."""
        if pod_id in self.pods:
            pod = self.pods[pod_id]
            pod.shutdown()
            del self.pods[pod_id]
            return True
        return False

    def spawn_team(self, size: int = 5) -> list[str]:
        """Spawn a full team of agents."""
        roles = ["chief", "coder", "researcher", "executor", "reviewer"]
        spawned = []
        for i, role in enumerate(roles[:size]):
            model = "llama3.1:70b" if role == "chief" else "llama3.1:8b"
            if role == "coder":
                model = "codellama"
            pod = self.create_pod(role, model)
            spawned.append(pod.pod_id)
        return spawned

