"""Phase 2: Multi-Agent Orchestrator Tests"""
from agent.core.orchestrator import Orchestrator
from agent.core.a2a_protocol import ChiefOfStaff, A2ATask, TaskStatus


def test_team_spawn():
    orch = Orchestrator()
    ids = orch.spawn_team(3)
    print(f"Spawned team: {len(ids)} agents")
    assert len(ids) == 3
    for i in ids:
        assert i.startswith("chief-") or i.startswith("coder-") or i.startswith("executor-")
    print("Team spawn: PASS")


def test_orchestrate_simple_task():
    orch = Orchestrator()
    # Simple task should route to executor
    result = orch.execute_task("create file test.txt with 'hello'")
    print(f"Task result: {result.get('status', 'unknown')}")
    assert result.get("status") == "completed"
    print("Orchestrate task: PASS")


def test_a2a_communication():
    chief = ChiefOfStaff()
    task = A2ATask(
        task_id="test-1",
        sender_id="user-1",
        receiver_id="coder-1",
        title="Write code",
        description="Write hello world",
    )
    # Send to coder
    agent = chief.sub_agents.get("coder-1")
    if not agent:
        agent = A2AAgent(agent_id="coder-1", role="coder")
        chief.sub_agents["coder-1"] = agent
    
    chief.delegate(task, "coder")
    print("A2A delegation: PASS")


if __name__ == "__main__":
    test_team_spawn()
    test_orchestrate_simple_task()
    test_a2a_communication()
    print("\nPhase 2 tests all passed!")

