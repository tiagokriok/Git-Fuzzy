package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const MaxRecent = 10

type Recent struct {
	Repositories []string `json:"repositories"`
}

func RecentPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get config dir: %w", err)
	}
	return filepath.Join(configDir, "gitf", "recent.json"), nil
}

func LoadRecent() (*Recent, error) {
	recentPath, err := RecentPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(recentPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Recent{Repositories: []string{}}, nil
		}
		return nil, fmt.Errorf("failed to read recent file: %w", err)
	}

	var recent Recent
	err = json.Unmarshal(data, &recent)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal recent data: %w", err)
	}
	return &recent, nil
}

func (r *Recent) Save() error {
	recentPath, err := RecentPath()
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(recentPath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal recent data: %w", err)
	}

	err = os.WriteFile(recentPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write recent file: %w", err)
	}
	return nil
}

func (r *Recent) Add(repoPath string) {
	var filtered []string
	for _, path := range r.Repositories {
		if path != repoPath {
			filtered = append(filtered, path)
		}
	}

	r.Repositories = append([]string{repoPath}, filtered...)

	if len(r.Repositories) > MaxRecent {
		r.Repositories = r.Repositories[:MaxRecent]
	}
}

func (r *Recent) GetRecent() []string {
	return r.Repositories
}
