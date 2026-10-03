package pods

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/google/uuid"
	"github.com/yourorg/vajra-bot/service/internal/db"

	log "github.com/sirupsen/logrus"
)

// Pod represents an agent pod instance
type Pod struct {
	ID       string
	Role     string
	Status   string
	Image    string
	CPU      string
	Memory   string
	Workspace string
	ContainerID string
}

// Manager handles pod lifecycle
type Manager struct {
	pods  map[string]*Pod
	db    *db.DB
	mu    sync.RWMutex
	created int
}

// NewManager creates a new pod manager
func NewManager(database *db.DB) *Manager {
	return &Manager{
		pods: make(map[string]*Pod),
		db:   database,
	}
}

// Create creates a new agent pod
func (m *Manager) Create(role string, image string, cpu string, memory string, workspace string) (*Pod, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	podID := uuid.New().String()

	// Validate workspace
	if workspace == "" {
		workspace = filepath.Join(os.TempDir(), "vajra-bot-workspace", podID)
	}
	if err := os.MkdirAll(workspace, 0755); err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}

	pod := &Pod{
		ID:       podID,
		Role:     role,
		Status:   "stopped",
		Image:    image,
		CPU:      cpu,
		Memory:   memory,
		Workspace: workspace,
	}

	// Create Docker container
	if err := m.createContainer(pod); err != nil {
		return nil, err
	}

	// Start container
	if err := m.startContainer(pod); err != nil {
		m.removeContainer(pod)
		return nil, err
	}

	m.pods[podID] = pod

	log.Infof("Created pod %s (role=%s, image=%s)", podID, role, image)
	return pod, nil
}

// createContainer creates a Docker container for the pod
func (m *Manager) createContainer(pod *Pod) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("create docker client: %w", err)
	}
	defer cli.Close()

	// Parse memory string
	memoryBytes, err := parseMemory(pod.Memory)
	if err != nil {
		return fmt.Errorf("parse memory: %w", err)
	}

	// Create container
	resp, err := cli.ContainerCreate(
		context.Background(),
		&container.Config{
			Image:         pod.Image,
			Hostname:       pod.ID,
			User:           "root:root",
			WorkingDir:     "/app",
			NetworkDisabled: true,
		},
		&container.HostConfig{
			CapDrop: []string{
				"ALL",
				"NET_ADMIN",
				"SYS_ADMIN",
				"SETUID",
				"SETGID",
			},
			Memory:         memoryBytes,
			MemorySwap:     memoryBytes * 2,
			CPUShares:      512,
			ReadonlyRootfs: true,
			SecurityOpt: []string{
				"no-new-privileges:true",
				"seccomp=unconfined",
			},
			LogConfig: container.LogConfig{
				Type: "json-file",
				Config: map[string]string{
					"max-size": "10m",
					"max-file": "3",
				},
			},
		},
		&networkingConfig{
			NetworkMode: container.NetworkMode("none"),
		},
		nil,
		nil,
		pod.ID,
	)
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}

	pod.ContainerID = resp.ID
	pod.Status = "stopped"
	return nil
}

// startContainer starts the pod container
func (m *Manager) startContainer(pod *Pod) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("create docker client: %w", err)
	}
	defer cli.Close()

	if err := cli.ContainerStart(context.Background(), pod.ContainerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("start container: %w", err)
	}

	pod.Status = "running"
	log.Infof("Started pod %s", pod.ContainerID)
	return nil
}

// Stop stops a pod
func (m *Manager) Stop(podID string) error {
	m.mu.Lock()
	pod, exists := m.pods[podID]
	m.mu.Unlock()
	if !exists {
		return fmt.Errorf("pod not found: %s", podID)
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("create docker client: %w", err)
	}
	defer cli.Close()

	// Stop with timeout
	timeout := 30
	if err := cli.ContainerStop(context.Background(), podID, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("stop container: %w", err)
	}

	pod.Status = "stopped"
	return nil
}

// GetPod returns a pod by ID
func (m *Manager) GetPod(podID string) (*Pod, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	pod, exists := m.pods[podID]
	if !exists {
		return nil, fmt.Errorf("pod not found: %s", podID)
	}
	return pod, nil
}

// ListPods returns all pods
func (m *Manager) ListPods() []*Pod {
	m.mu.RLock()
	defer m.mu.RUnlock()
	pods := make([]*Pod, 0, len(m.pods))
	for _, pod := range m.pods {
		pods = append(pods, pod)
	}
	return pods
}

// GetPodLogs returns pod logs
func (m *Manager) GetPodLogs(podID string) (string, error) {
	pod, err := m.GetPod(podID)
	if err != nil {
		return "", err
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf("create docker client: %w", err)
	}
	defer cli.Close()

	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       "50",
	}

	reader, err := cli.ContainerLogs(context.Background(), pod.ContainerID, options)
	if err != nil {
		return "", fmt.Errorf("get logs: %w", err)
	}
	defer reader.Close()

	var buf []byte
	_, err = io.Copy(&buf, reader)
	if err != nil {
		return "", fmt.Errorf("read logs: %w", err)
	}

	return string(buf), nil
}

// ExecInPod executes a command in the pod
func (m *Manager) ExecInPod(podID string, cmd []string) (string, error) {
	pod, err := m.GetPod(podID)
	if err != nil {
		return "", err
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf("create docker client: %w", err)
	}
	defer cli.Close()

	execConfig := types.ExecConfig{
		Container:  pod.ContainerID,
		Cmd:        cmd,
		WorkingDir: "/app",
		User:       "root:root",
	}

	execResp, err := cli.ContainerExecCreate(context.Background(), pod.ContainerID, execConfig)
	if err != nil {
		return "", fmt.Errorf("create exec: %w", err)
	}

	attachResp, err := cli.ContainerExecAttach(context.Background(), execResp.ID, types.ExecStartCheck{})
	if err != nil {
		return "", fmt.Errorf("attach exec: %w", err)
	}
	defer attachResp.Conn.Close()

	var buf []byte
	_, err = io.Copy(&buf, attachResp.Reader)
	if err != nil {
		return "", fmt.Errorf("read exec output: %w", err)
	}

	return string(buf), nil
}

// cleanupWorkspace removes the pod workspace after stopping
func (m *Manager) cleanupWorkspace(pod *Pod) {
	if pod.Workspace != "" {
		if err := os.RemoveAll(pod.Workspace); err != nil {
			log.Warnf("failed to remove workspace %s: %v", pod.Workspace, err)
		}
	}
}

// parseMemory converts a memory string to bytes (e.g. "4g" -> 4294967296)
func parseMemory(s string) (int64, error) {
	s = os.ExpandEnv(s)
	u, err := url.Parse(s)
	if err != nil {
		return 0, fmt.Errorf("parse memory: %w", err)
	}

	// Simple unit conversion
	multiplier := int64(1)
	switch u.Scheme {
	case "m", "mb":
		multiplier = 1024 * 1024
	case "g", "gb":
		multiplier = 1024 * 1024 * 1024
	case "k", "kb":
		multiplier = 1024
	default:
		i, err := fmt.Sscanf(u.Opaque, "%d", &multiplier)
		if err != nil || i == 0 {
			// Assume bytes
			multiplier = 1
		}
	}

	var amount int64
	fmt.Sscanf(u.Opaque, "%d", &amount)
	return amount * multiplier, nil
}
