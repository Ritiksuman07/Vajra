package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

// Models creates the `models` command.
func Models() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "models",
		Short: "List available models from all providers",
		Long: `List models available from configured providers.
Shows both local models (from Ollama, LM Studio, llama.cpp) 
and cloud models (if providers are configured via /connect).`,
		Example: strings.Join([]string{
			`  vajra-bot models`,
			`  vajra-bot models --provider ollama`,
			`  vajra-bot models --json`,
		}, "\n"),
		RunE: func(cmd *cobra.Command, args []string) error {
			provider, _ := cmd.Flags().GetString("provider")
			jsonOutput, _ := cmd.Flags().GetBool("json")

			serviceURL := "http://localhost:4096"
			if envURL := os.Getenv("vajra_BOT_URL"); envURL != "" {
				serviceURL = envURL
			}

			if provider != "" {
				return listProviderModels(serviceURL, provider, jsonOutput)
			}
			return listAllModels(serviceURL, jsonOutput)
		},
	}

	cmd.Flags().StringP("provider", "p", "", "Filter by provider name (e.g., ollama, openai)")
	cmd.Flags().BoolP("json", "j", false, "Output as JSON")

	return cmd
}

// ModelInfo represents model metadata.
type ModelInfo struct {
	Name         string `json:"name"`
	Provider     string `json:"provider"`
	Type         string `json:"type"` // local, cloud
	Size         string `json:"size"`
	Context      int    `json:"context"`
	Capabilities []string `json:"capabilities"`
	Status       string `json:"status"`
}

// listAllModels gets models from all providers.
func listAllModels(url string, jsonOutput bool) error {
	resp, err := http.Get(url + "/models")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		Models []ModelInfo `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result.Models)
	}

	fmt.Println("Available Models:")
	fmt.Println("-----------------")
	for _, m := range result.Models {
		status := "✓ available"
		if m.Status == "configuring" {
			status = "⚠ configuring"
		}
		fmt.Printf("%-20s %-12s %-8s %d ctx %v\n", m.Name, m.Provider, m.Size, m.Context, m.Capabilities)
	}
	return nil
}

// listProviderModels gets models from a specific provider.
func listProviderModels(url, provider string, jsonOutput bool) error {
	endpoint := fmt.Sprintf("%s/models?provider=%s", url, provider)
	resp, err := http.Get(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		Models []ModelInfo `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result.Models)
	}

	fmt.Printf("Models from %s:\n", provider)
	fmt.Println("-----------------")
	for _, m := range result.Models {
		fmt.Printf("- %s (%s) %s %d ctx\n", m.Name, m.Size, strings.Join(m.Capabilities, ", "), m.Context)
	}
	return nil
}

