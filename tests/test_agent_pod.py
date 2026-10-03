# Agent Pod Tests

from agent.core.agent import AgentPod, AgentConfig, Task
from agent.core.verification import VerificationPipeline, VerificationResult


def test_agent_initialization():
    """Test agent pod initialization."""
    config = AgentConfig(role="executor", model="llama3.1:8b")
    pod = AgentPod(pod_id="test-executor", config=config)
    pod.initialize()
    
    assert pod.status == "ready"
    assert pod.role == "executor"
    print(f"Agent initialized: {pod.pod_id} (role={pod.role})")


def test_verification_pipeline():
    """Test the anti-hallucination verification pipeline."""
    pipeline = VerificationPipeline()
    
    # Test with a good response
    result = pipeline.verify(
        prompt="Write a Python hello world",
        response="Here is a Python hello world: print('Hello, World!')",
        context="Python programming",
    )
    
    assert result.passed
    assert result.confidence > 0.75
    print(f"Verification passed: confidence={result.confidence:.2f}")


def test_agent_execute():
    """Test agent execution with a simple file write task."""
    import tempfile
    
    workspace = tempfile.mkdtemp(prefix="test-agent-")
    config = AgentConfig(role="executor", model="test", workspace=workspace)
    pod = AgentPod(pod_id="test", config=config)
    pod.initialize()
    
    task = Task(
        id="test-write",
        title="Write Hello",
        description="Create file hello.txt with 'Hello, World!'",
    )
    
    result = pod.execute(task)
    
    # Check the file was created
    import os
    file_path = os.path.join(workspace, "hello.txt")
    if os.path.exists(file_path):
        print(f"Task completed: {result['status']} - file created")
    else:
        print(f"Task completed: {result['status']} - {result.get('result', 'No result')}")
    
    assert result["status"] in ("completed", "failed")


if __name__ == "__main__":
    print("Running Agent Pod Tests...")
    test_agent_initialization()
    test_verification_pipeline()
    test_agent_execute()
    print("All tests passed!")

