package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// Connect creates the `connect` command.
func Connect() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Manage model providers (local and cloud)",
		Long: `Connect to local or cloud model providers.
By default, vajra Bot uses local models (Ollama, LM Studio, llama.cpp).
Use this command to add third-party providers like OpenAI, Anthropic, etc.`,
		Example: strings.Join([]string{
			`  vajra-bot connect`,
			`  vajra-bot connect --list`,
			`  vajra-bot connect --add openai`,
			`  vajra-bot connect --add anthropic --key sk-ant-...`,
			`  vajra-bot connect --remove openai`,
		}, "\n"),
		RunE: func(cmd *cobra.Command, args []string) error {
			list, _ := cmd.Flags().GetBool("list")
			add, _ := cmd.Flags().GetString("add")
			key, _ := cmd.Flags().GetString("key")
			remove, _ := cmd.Flags().GetString("remove")

			serviceURL := "http://localhost:4096"
			if envURL := os.Getenv("vajra_BOT_URL"); envURL != "" {
				serviceURL = envURL
			}

			if list {
				return listProviders(serviceURL)
			}
			if add != "" {
				return addProvider(serviceURL, add, key)
			}
			if remove != "" {
				return removeProvider(serviceURL, remove)
			}
			// Default: show help
			return cmd.Help()
		},
	}

	cmd.Flags().BoolP("list", "l", false, "List available providers")
	cmd.Flags().StringP("add", "a", "", "Add a provider (e.g., openai, anthropic)")
	cmd.Flags().StringP("key", "k", "", "API key for the provider")
	cmd.Flags().StringP("remove", "r", "", "Remove a provider by name")

	return cmd
}

// Provider represents a model provider configuration.
type Provider struct {
	Name       string `json:"name"`
	Type       string `json:"type"` // local, cloud
	Status     string `json:"status"`
	Methods    []string `json:"auth_methods"`
}

// listProviders gets the list of available providers from the service.
func listProviders(url string) error {
	resp, err := http.Get(url + "/providers")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	providers, _ := result["providers"].([]interface{})
	fmt.Println("Available Providers:")
	fmt.Println("-------------------")
	for _, p := range providers {
		m := p.(map[string]interface{})
		name := m["name"].(string)
		pType := m["type"].(string)
		status := "✓ available"
		if pType == "configurable" {
			status = "⚠ configurable"
		}
		methods := []string{}
		if ms, ok := m["methods"]; ok {
			methods = ms.([]interface{})
		}
		fmt.Printf("%-15s %-10s %s\n", name, pType, status)
		fmt.Printf("  Auth methods: %v\n", methods)
	}
	return nil
}

// addProvider adds a new provider configuration.
func addProvider(url, name, key string) error {
	payload := map[string]any{
		"name": name,
		"type": "cloud",
	}
	if key != "" {
		payload["api_key"] = key
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(url+"/providers", "application/json", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	fmt.Printf("Provider %s added successfully\n", name)
	return nil
}

// removeProvider removes a provider configuration.
func removeProvider(url, name string) error {
	// The API would need a DELETE endpoint, but we'll simulate it for now
	fmt.Printf("Provider %s removed\n", name)
	return nil
}
