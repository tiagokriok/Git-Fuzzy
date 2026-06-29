package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/tiagokriok/Git-Fuzzy/internal/config"
	"github.com/tiagokriok/Git-Fuzzy/internal/history"
	"github.com/tiagokriok/Git-Fuzzy/internal/scanner"
	"github.com/tiagokriok/Git-Fuzzy/internal/theme"
	"github.com/tiagokriok/Git-Fuzzy/internal/ui"
)

// Version information (injected at build time via ldflags)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "source"
)

func getVersionInfo() string {
	return fmt.Sprintf("gitf version %s\ncommit: %s\nbuilt at: %s\nbuilt by: %s",
		version, commit, date, builtBy)
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "gitf",
		Short: "Git Fuzzy - Interactive repository selector",
		Long: `Git Fuzzy (gitf) is a fast, interactive git repository finder.

It scans configured directories for git repositories and lets you
quickly select one using fuzzy search. Selected repositories open
automatically in your preferred editor.`,
		Version:      getVersionInfo(),
		RunE:         runTUI,
		SilenceUsage: true,
	}

	// Add --setup flag
	var setupFlag bool
	rootCmd.Flags().BoolVarP(&setupFlag, "setup", "s", false,
		"Force configuration wizard to run")

	// Handle --setup before main RunE
	rootCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if setupFlag {
			if err := handleSetup(cmd, args); err != nil {
				return err
			}
			os.Exit(0)
		}
		return nil
	}

	// Custom version template for cleaner output
	rootCmd.SetVersionTemplate(`{{.Version}}` + "\n")

	themeCmd := &cobra.Command{
		Use:   "theme",
		Short: "Theme utilities",
	}
	syncOmarchyCmd := &cobra.Command{
		Use:   "sync-omarchy",
		Short: "Sync the current Omarchy theme into gitf cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runThemeSyncOmarchy(cmd.OutOrStdout(), cmd.ErrOrStderr(), theme.SyncOmarchy)
		},
	}
	themeCmd.AddCommand(syncOmarchyCmd)
	rootCmd.AddCommand(themeCmd)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runTUI(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			setupThemeCfg := config.ThemeConfig{Mode: string(config.ThemeModeAuto)}
			setupTheme, setupWarning := theme.Load(setupThemeCfg)
			cfg, err = ui.RunSetup(setupTheme, setupWarning, setupThemeCfg)
			if err != nil {
				return fmt.Errorf("setup failed: %w", err)
			}
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
		} else {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	appTheme, themeWarning := theme.Load(cfg.GetThemeConfig())

	repos, err := scanner.Scan(cfg.SearchPaths)
	if err != nil {
		return fmt.Errorf("failed to scan repositories: %w", err)
	}

	recents, err := history.LoadRecent()
	if err == nil {
		repos = scanner.ReorderByRecent(repos, recents)
	}

	if len(repos) == 0 {
		return fmt.Errorf("no repositories found in search paths")
	}

	selected, err := ui.Run(repos, cfg, appTheme, themeWarning)
	if err != nil {
		return fmt.Errorf("failed to run UI: %w", err)
	}

	if selected == nil {
		return nil // User cancelled
	}

	if err := openRepositoryInEditor(cfg, selected.Path); err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	recent, err := history.LoadRecent()
	if err == nil {
		recent.Add(selected.Path)
		recent.Save()
	}

	return nil
}

func handleSetup(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to load config: %w", err)
	}

	var themeConfig config.ThemeConfig
	if cfg != nil {
		themeConfig = cfg.GetThemeConfig()
	} else {
		themeConfig = config.ThemeConfig{Mode: string(config.ThemeModeAuto)}
	}

	setupTheme, themeWarning := theme.Load(themeConfig)
	styles := ui.NewStyles(setupTheme)

	if cfg != nil {
		fmt.Println()
		fmt.Println(styles.Title.Render("Current configuration:"))
		fmt.Println(styles.Muted.Render(fmt.Sprintf("  Editor: %s", cfg.Editor)))
		fmt.Println(styles.Muted.Render(fmt.Sprintf("  Search Paths: %s", strings.Join(cfg.SearchPaths, ", "))))
		if themeWarning != nil {
			fmt.Println()
			fmt.Println(styles.Error.Render("Warning: " + themeWarning.Message))
		}
		fmt.Println()
		fmt.Println(styles.Muted.Render("Press Enter to setup or Ctrl+C to cancel..."))
		fmt.Scanln()
	}

	newCfg, err := ui.RunSetup(setupTheme, themeWarning, themeConfig)
	if err != nil {
		return fmt.Errorf("setup cancelled: %w", err)
	}

	newCfg.Theme = themeConfig

	if err := newCfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println(lipgloss.NewStyle().Foreground(styles.Success).Render("\n✓ Configuration saved successfully!"))
	return nil
}

func runThemeSyncOmarchy(stdout io.Writer, stderr io.Writer, syncFn func() (theme.Cache, error)) error {
	cache, err := syncFn()
	if err != nil {
		fmt.Fprintf(stderr, "failed to sync Omarchy theme: %v\n", err)
		return err
	}
	cachePath, pathErr := theme.CachePath()
	if pathErr != nil {
		fmt.Fprintf(stdout, "Synced Omarchy theme %q\n", cache.Name)
		return nil
	}
	fmt.Fprintf(stdout, "Synced Omarchy theme %q to %s\n", cache.Name, cachePath)
	return nil
}

// isTerminalEditor checks if the editor is terminal-based
func isTerminalEditor(editor string) bool {
	terminalEditors := []string{"nvim", "vim", "vi", "nano", "helix", "hx", "emacs", "micro", "ne", "joe", "jed"}
	editorBase := strings.ToLower(editor)
	// Handle full paths like /usr/bin/nvim
	if idx := strings.LastIndex(editorBase, "/"); idx != -1 {
		editorBase = editorBase[idx+1:]
	}
	for _, te := range terminalEditors {
		if editorBase == te {
			return true
		}
	}
	return false
}

// buildTerminalCommand builds the correct command to run editor in a new terminal window
// Different terminals have different syntax for executing commands
func buildTerminalCommand(terminal, editor, path string) *exec.Cmd {
	switch terminal {
	// Terminals using -e flag for command execution
	case "ghostty", "alacritty", "xterm", "urxvt", "rxvt", "terminology":
		return exec.Command(terminal, "-e", editor, path)

	// Kitty uses special syntax
	case "kitty":
		return exec.Command(terminal, "--directory", path, editor, path)

	// WezTerm uses start subcommand
	case "wezterm":
		return exec.Command(terminal, "start", "--cwd", path, editor, path)

	// Konsole uses -e with workdir
	case "konsole":
		return exec.Command(terminal, "--workdir", path, "-e", editor, path)

	// GNOME Terminal uses -- to separate args
	case "gnome-terminal":
		return exec.Command(terminal, "--working-directory", path, "--", editor, path)

	// Tilix uses -e
	case "tilix":
		return exec.Command(terminal, "--working-directory", path, "-e", editor+" "+path)

	// xdg-terminal-exec and x-terminal-emulator handle syntax automatically
	case "xdg-terminal-exec", "x-terminal-emulator":
		return exec.Command(terminal, editor, path)

	// Default: try -e flag (most common)
	default:
		return exec.Command(terminal, "-e", editor, path)
	}
}

func openRepositoryInEditor(cfg *config.Config, path string) error {
	editor := cfg.Editor

	// If editor is terminal-based, open in a new terminal window
	if isTerminalEditor(editor) {
		terminal := cfg.GetTerminal()
		if terminal != "" {
			cmd := buildTerminalCommand(terminal, editor, path)
			cmd.Dir = path

			// Detach the child process so it survives parent exit
			// This is critical for floating window launchers (Hyprland, etc.)
			detachProcess(cmd)

			if err := cmd.Start(); err != nil {
				return err
			}

			// Small delay to ensure process is fully spawned before parent exits
			time.Sleep(50 * time.Millisecond)
			return nil
		}
	}

	// GUI editors open directly
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
