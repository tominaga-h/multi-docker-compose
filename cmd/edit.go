package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"mdc/internal/config"

	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit <config-name>",
	Short: "Open a config file in your editor ($EDITOR or vim)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path, err := config.ResolveConfigPath(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := openEditor(path); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}

func openEditor(filePath string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	c := exec.Command(editor, filePath)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		return fmt.Errorf("failed to open editor %q: %w", editor, err)
	}
	return nil
}
