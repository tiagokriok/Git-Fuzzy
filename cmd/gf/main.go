package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/tiagokriok/gf/internal/config"
	"github.com/tiagokriok/gf/internal/scanner"
	"github.com/tiagokriok/gf/internal/ui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg, err = setupWizard()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error setting up configuration: %v\n", err)
				os.Exit(1)
			}

			err = cfg.Save()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error saving configuration: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
	}

	repos, err := scanner.Scan(cfg.SearchPaths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning repositories: %v\n", err)
		os.Exit(1)
	}

	if len(repos) == 0 {
		fmt.Fprintf(os.Stderr, "No repositories found\n")
		os.Exit(1)
	}

	selected, err := ui.Run(repos)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running UI: %v\n", err)
		os.Exit(1)
	}

	if selected == nil {
		os.Exit(0)
	}

	err = openRepositoryInEditor(cfg.Editor, selected.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening repository in editor: %v\n", err)
		os.Exit(1)
	}

}

func openRepositoryInEditor(editor string, path string) error {
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func setupWizard() (*config.Config, error) {
	return ui.RunSetup()
}
