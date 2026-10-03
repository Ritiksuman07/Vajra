package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// Agents creates the `agents` command.
func Agents() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Manage agent pods (Chief of Staff, Coder, Researcher, etc.)",
		Long: `List, create, and manage agent pods for multi-agent coordination.
Agent pods work together on complex tasks with a Chief of Staff delegating work.`,
		Example: strings.Join([]string{
			`  vajra-bot agents list`,
			`  vajra-bot agents create --role chief --model llama3.1:70b`,
			`  vajra-bot agents create --role coder --model codellama`,
			`  vajra-bot agents stop <pod-id>`,
		}, "\n"),
		RunE: func(cmd *cobra.Command, args []string) error {
			action, _ := cmd.Flags().GetString("action")

			serviceURL := "http://localhost:4096"
			if envURL := os.Getenv("vajra_BOT_URL"); envURL != "" {
				serviceURL = envURL
			}

			switch action {
			case "list":
				return listAgentPods(serviceURL)
			case "create":
				role, _ := cmd.Flags().GetString("role")
				model, _ := cmd.Flags().GetString("model")
				return createAgentPod(serviceURL, role, model)
			case "stop":
				if len(args) < 1 {
					return fmt.Errorf("pod ID required for stop action")
				}
				return stopAgentPod(serviceURL, args[0])
			default:
				return cmd.Help()
			}
		},
	}

	cmd.Flags().StringP("action", "a", "list", "Action: list, create, stop, status")
	cmd.Flags().StringP("role", "r", "agent", "Agent role: chief, coder, researcher, executor, reviewer")
	cmd.Flags().StringP("model", "m", "", "Model to use for this agent")

	return cmd
}

// AgentPodInfo represents agent pod information.
type AgentPodInfo struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	Model     string `json:"model"`
	Workspace string `json:"workspace"`
	Created   string `json:"created_at"`
}

// listAgentPods lists running agent pods.
func listAgentPods(url string) error {
	resp, err := http.Get(url + "/pods")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		Pods  []AgentPodInfo `json:"pods"`
		Total int `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	fmt.Println("Agent Pods:")
	fmt.Println("------------")
	for _, pod := range result.Pods {
		fmt.Printf("%-8s %-8s %-10s %-20s\n", pod.ID[:8], pod.Role, pod.Status, pod.Model)
	}
	fmt.Printf("\nTotal: %d pods\n", result.Total)
	return nil
}

// createAgentPod creates a new agent pod.
func createAgentPod(url, role, model string) error {
	payload := fmt.Sprintf(`{"role":"%s","image":"vajra-bot-agent:latest","cpu":"2","memory":"4g","workspace":"%s"}`,
		role, fmt.Sprintf("/workspace/%s", role))

	resp, err := http.Post(url+"/pods", "application/json", strings.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	fmt.Printf("Agent pod created: %s (role=%s)\n", result.ID, role)
	return nil
}

// stopAgentPod stops an agent pod.
func stopAgentPod(url, podID string) error {
	req, err := http.NewRequest("DELETE", url+"/pods/"+podID, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Printf("Agent pod %s stopped\n", podID)
	return nil
}

