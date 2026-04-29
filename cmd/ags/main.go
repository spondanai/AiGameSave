package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spondanai/aigamesave/internal/adapters"
	"github.com/spondanai/aigamesave/internal/usecase"
)

const modulePath = "github.com/spondanai/aigamesave"
const updateCheckInterval = 24 * time.Hour

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
		adapterName, _ := cmd.Flags().GetString("adapter")

		cwd, err := os.Getwd()
		if err != nil {
			fmt.Println("Error getting current directory:", err)
			os.Exit(1)
		}

		err = usecase.SaveGameWithAdapter(cwd, adapterName)
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
	saveCmd.Flags().String("adapter", "", fmt.Sprintf("Force a specific adapter (%s)", strings.Join(adapters.AdapterNames(), ", ")))
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

	latest, err := latestModuleInfoCached()
	if err != nil || latest.Version == "" || isCurrentBuild(latest) {
		return
	}

	fmt.Printf("Warning: AGS %s is available (current: %s). Run `ags self-update` to update adapters.\n", latest.Version, current)
}

type moduleInfo struct {
	Version string `json:"Version"`
	Origin  struct {
		Hash string `json:"Hash"`
	} `json:"Origin"`
}

type updateCache struct {
	CheckedAt     time.Time `json:"checked_at"`
	LatestVersion string    `json:"latest_version"`
	LatestHash    string    `json:"latest_hash"`
}

func latestModuleInfoCached() (moduleInfo, error) {
	if cached, ok := readUpdateCache(); ok && time.Since(cached.CheckedAt) < updateCheckInterval {
		return moduleInfoFromCache(cached), nil
	}

	latest, err := latestModuleInfo()
	if err != nil {
		if cached, ok := readUpdateCache(); ok {
			return moduleInfoFromCache(cached), nil
		}
		return moduleInfo{}, err
	}

	_ = writeUpdateCache(updateCache{
		CheckedAt:     time.Now(),
		LatestVersion: latest.Version,
		LatestHash:    latest.Origin.Hash,
	})
	return latest, nil
}

func moduleInfoFromCache(cache updateCache) moduleInfo {
	var info moduleInfo
	info.Version = cache.LatestVersion
	info.Origin.Hash = cache.LatestHash
	return info
}

func latestModuleInfo() (moduleInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "list", "-m", "-json", modulePath+"@latest")
	cmd.Env = append(os.Environ(), "GOPROXY=direct")
	out, err := cmd.Output()
	if err != nil {
		return moduleInfo{}, err
	}

	var mod moduleInfo
	if err := json.Unmarshal(out, &mod); err != nil {
		return moduleInfo{}, err
	}
	return mod, nil
}

func readUpdateCache() (updateCache, bool) {
	path, err := updateCachePath()
	if err != nil {
		return updateCache{}, false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return updateCache{}, false
	}

	var cache updateCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return updateCache{}, false
	}
	if cache.LatestVersion == "" {
		return updateCache{}, false
	}
	return cache, true
}

func writeUpdateCache(cache updateCache) error {
	path, err := updateCachePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func updateCachePath() (string, error) {
	if path := os.Getenv("AGS_UPDATE_CACHE"); path != "" {
		return path, nil
	}
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "aigamesave", "update_check.json"), nil
}

func isCurrentBuild(latest moduleInfo) bool {
	current := currentVersion()
	if current != "dev" && latest.Version == current {
		return true
	}

	revision := currentRevision()
	return revision != "" && latest.Origin.Hash != "" && strings.HasPrefix(latest.Origin.Hash, revision)
}

func currentRevision() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			return setting.Value
		}
	}
	return ""
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
