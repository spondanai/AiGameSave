package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spondanai/aigamesave/internal/usecase"
)

var rootCmd = &cobra.Command{
	Use:   "ags",
	Short: "AiGameSave (AGS) - Save and load AI CLI context",
	Long:  `AGS extracts the current context (history + git status) and saves it to a YAML file to resume your session later without wasting tokens.`,
}

var saveCmd = &cobra.Command{
	Use:   "save",
	Short: "Save the current AI CLI session",
	Run: func(cmd *cobra.Command, args []string) {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Println("Error getting current directory:", err)
			os.Exit(1)
		}

		err = usecase.SaveGame(cwd)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
	},
}

var loadCmd = &cobra.Command{
	Use:   "load",
	Short: "Load the saved AI CLI session",
	Run: func(cmd *cobra.Command, args []string) {
		stdoutFlag, _ := cmd.Flags().GetBool("stdout")
		
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Println("Error getting current directory:", err)
			os.Exit(1)
		}

		prompt, err := usecase.LoadGame(cwd, !stdoutFlag)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		
		if stdoutFlag {
			fmt.Print(prompt)
		}
	},
}

func init() {
	loadCmd.Flags().Bool("stdout", false, "Print to stdout instead of copying to clipboard")
	rootCmd.AddCommand(saveCmd, loadCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
