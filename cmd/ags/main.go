package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"
	"time"

	"github.com/spf13/cobra"
	"github.com/spondanai/aigamesave/internal/usecase"
)

const modulePath = "github.com/spondanai/aigamesave"

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "ags",
	Short:   "AiGameSave (AGS) - Save and load AI CLI context",
	Long:    `AGS extracts the current context (history + git status) and saves it to a YAML file to resume your session later without wasting tokens.`,
	Version: currentVersion(),
}

var saveCmd = &cobra.Command{
	Use:   "save",
	Short: "Save the current AI CLI session",
	Run: func(cmd *cobra.Command, args []string) {
		warnIfUpdateAvailable()

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
		warnIfUpdateAvailable()

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

var selfUpdateCmd = &cobra.Command{
	Use:   "self-update",
	Short: "Install the latest AGS version",
	Run: func(cmd *cobra.Command, args []string) {
		if err := selfUpdate(); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
	},
}

func init() {
	loadCmd.Flags().Bool("stdout", false, "Print to stdout instead of copying to clipboard")
	rootCmd.AddCommand(saveCmd, loadCmd, selfUpdateCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func currentVersion() string {
	if version != "" && version != "dev" {
		return version
	}

	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return "dev"
	}
	return info.Main.Version
}

func warnIfUpdateAvailable() {
	if os.Getenv("AGS_SKIP_UPDATE_CHECK") == "1" {
		return
	}

	current := currentVersion()
	if current == "dev" {
		return
	}

	latest, err := latestModuleVersion()
	if err != nil || latest == "" || latest == current {
		return
	}

	fmt.Printf("Warning: AGS %s is available (current: %s). Run `ags self-update` to update adapters.\n", latest, current)
}

func latestModuleVersion() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "list", "-m", "-json", modulePath+"@latest")
	cmd.Env = append(os.Environ(), "GOPROXY=direct")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	var mod struct {
		Version string
	}
	if err := json.Unmarshal(out, &mod); err != nil {
		return "", err
	}
	return mod.Version, nil
}

func selfUpdate() error {
	fmt.Println("Installing latest AGS...")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "install", modulePath+"/cmd/ags@latest")
	cmd.Env = append(os.Environ(), "GOPROXY=direct")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run go install: %w", err)
	}

	fmt.Println("AGS updated successfully.")
	return nil
}
