package scanner

import (
	"os"
	"path/filepath"
	"sort"
)

type Repository struct {
	Name string
	Path string
}

var ignoredDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	".git":         true,
	".idea":        true,
	".config":      true,
	".cache":       true,
	".vscode":      true,
	"venv":         true,
	"venv3":        true,
	".venv":        true,
	".venv3":       true,
	"target":       true,
}

func shouldIgnore(name string) bool {
	return ignoredDirs[name]
}

func isGitRepository(path string) bool {
	gitPath := filepath.Join(path, ".git")
	info, err := os.Stat(gitPath)
	return err == nil && info.IsDir()
}

func Scan(searchPaths []string) ([]Repository, error) {
	var repos []Repository
	seen := make(map[string]bool)

	for _, searchPath := range searchPaths {
		info, err := os.Stat(searchPath)
		if err != nil {
			continue
		}

		if !info.IsDir() {
			continue
		}

		absPath, err := filepath.Abs(searchPath)
		if err != nil {
			continue
		}

		err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if !info.IsDir() {
				return nil
			}

			dirName := filepath.Base(path)

			if shouldIgnore(dirName) {
				return filepath.SkipDir
			}

			if isGitRepository(path) {
				repoName := dirName

				if !seen[path] {
					seen[path] = true
					repos = append(repos, Repository{
						Name: repoName,
						Path: path,
					})
				}

				return filepath.SkipDir
			}

			return nil
		})

		if err != nil {
			continue
		}

	}

	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Name < repos[j].Name
	})

	return repos, nil
}
