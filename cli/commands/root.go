package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Root creates the root command for the CLI.
func Root() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vajra-bot",
		Short: "Local-first open-source vajra Bot clone",
		Long: `vajra Bot is a team of always-on AI agents that work locally.
Use it completely offline (default) or connect third-party providers
with /connect (OpenAI, OpenRouter, Anthropic, etc.).

Built with anti-hallucination verification, multi-agent coordination,
and full local model support (Ollama, LM Studio, llama.cpp).`,
	}

	// Add subcommands
	cmd.AddCommand(Run())
	cmd.AddCommand(Connect())
	cmd.AddCommand(Models())
	cmd.AddCommand(Agents())
	cmd.AddCommand(Service())

	return cmd
}

// Execute runs the CLI.
func Execute() error {
	cmd := Root()
	return cmd.Execute()
}

func main() {
	if err := Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

