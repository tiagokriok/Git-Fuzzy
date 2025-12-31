package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
	"github.com/tiagokriok/gf/internal/scanner"
)

var selectedRepository *scanner.Repository

type Model struct {
	repositories []scanner.Repository
	filtered     []scanner.Repository
	searchInput  string
	selectedIdx  int
	width        int
	height       int
	err          error
}

func NewModel(repos []scanner.Repository) Model {
	return Model{
		repositories: repos,
		filtered:     repos,
		selectedIdx:  0,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m Model) View() string {
	var s string
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))

	s += headerStyle.Render("🔍 Git Finder") + "\n"
	s += strings.Repeat("─", 40) + "\n\n"

	s += fmt.Sprintf("Search: %s\n\n", m.searchInput)

	if len(m.filtered) == 0 {
		s += "No repositories found.\n"
	} else {
		selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
		for i, repo := range m.filtered {
			line := fmt.Sprintf("%s (%s)", repo.Name, repo.Path)

			if i == m.selectedIdx {
				s += selectedStyle.Render("▶ "+line) + "\n"
			} else {
				s += "  " + line + "\n"
			}
		}
	}

	s += "\n" + strings.Repeat("─", 40) + "\n"
	s += "↑/↓: navigate | Enter: select | Esc: exit\n"

	return s
}

func (m *Model) updateFiltered() {
	if m.searchInput == "" {
		m.filtered = m.repositories
		return
	}

	names := make([]string, len(m.repositories))
	for i, repo := range m.repositories {
		names[i] = repo.Name
	}

	matches := fuzzy.Find(m.searchInput, names)

	m.filtered = make([]scanner.Repository, len(matches))
	for i, match := range matches {
		m.filtered[i] = m.repositories[match.Index]
	}
}

func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		selectedRepository = nil
		return m, tea.Quit
	case "up", "shift+tab":
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
		return m, nil
	case "down", "tab":
		if m.selectedIdx < len(m.filtered)-1 {
			m.selectedIdx++
		}
		return m, nil
	case "enter":
		if len(m.filtered) > 0 {
			selected := m.filtered[m.selectedIdx]
			selectedRepository = &selected
			return m, tea.Quit
		}
		return m, nil
	case "backspace":
		if len(m.searchInput) > 0 {
			m.searchInput = m.searchInput[:len(m.searchInput)-1]
			m.updateFiltered()
			m.selectedIdx = 0
		}
		return m, nil
	default:
		m.searchInput += msg.String()
		m.updateFiltered()
		m.selectedIdx = 0
		return m, nil
	}
}

func GetSelectedRepository() *scanner.Repository {
	return selectedRepository
}

func Run(repos []scanner.Repository) (*scanner.Repository, error) {
	selectedRepository = nil

	model := NewModel(repos)

	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		return nil, fmt.Errorf("TUI Error: %w", err)
	}

	return selectedRepository, nil
}
