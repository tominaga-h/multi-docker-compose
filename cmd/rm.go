package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"mdc/internal/config"

	"github.com/spf13/cobra"
)

var rmForce bool

var rmCmd = &cobra.Command{
	Use:   "rm <config-name>",
	Short: "Remove a config YAML file from ~/.config/mdc/",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		path, err := config.ResolveConfigPath(name)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		displayPath := config.ContractHome(path)

		if !rmForce {
			fmt.Printf("Are you sure you want to remove the file '%s'? [y/n]: ", displayPath)
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(answer)
			if answer != "y" {
				fmt.Println("Interrupted")
				return
			}
		}

		if _, err := config.RemoveDefaultConfig(name); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("Removed %s\n", displayPath)
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
	rmCmd.Flags().BoolVarP(&rmForce, "force", "f", false, "Skip confirmation prompt")
}
