package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tiagokriok/Git-Fuzzy/internal/config"
)

type SetupModel struct {
	step               int
	editor             textinput.Model
	paths              textinput.Model
	tmuxActionIdx      int
	tmuxDefaultActions []config.TmuxOpenAction
	completed          *config.Config
	err                error
}

func RunSetup() (*config.Config, error) {
	editorInput := textinput.New()
	editorInput.Placeholder = "e.g., vim, nvim, code, zed"
	editorInput.SetValue("nvim")
	editorInput.Focus()

	pathsInput := textinput.New()
	defaultPaths := ""
	pathsInput.Placeholder = "e.g., ~/dev, ~/projects"
	pathsInput.SetValue(defaultPaths)

	model := SetupModel{
		step:               0,
		editor:             editorInput,
		paths:              pathsInput,
		tmuxDefaultActions: config.TmuxOpenActionOptions(),
	}

	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	setupModel := finalModel.(SetupModel)
	if setupModel.err != nil {
		return nil, setupModel.err
	}

	return setupModel.completed, nil
}

func (m SetupModel) Init() tea.Cmd {
	return nil
}

func (m SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.err = fmt.Errorf("setup cancelled")
			return m, tea.Quit

		case "enter":
			if m.step == 0 {
				if m.editor.Value() == "" {
					m.err = fmt.Errorf("editor cannot be empty")
					return m, tea.Quit
				}
				m.step = 1
				m.paths.Focus()
				m.editor.Blur()
				return m, nil
			}

			if m.step == 1 {
				if m.paths.Value() == "" {
					m.err = fmt.Errorf("paths cannot be empty")
					return m, tea.Quit
				}
				m.step = 2
				m.paths.Blur()
				return m, nil
			}

			homeDir, _ := os.UserHomeDir()
			var searchPaths []string
			for path := range strings.SplitSeq(m.paths.Value(), ",") {
				path = strings.TrimSpace(path)
				if path != "" {
					if strings.HasPrefix(path, "~") {
						path = strings.Replace(path, "~", homeDir, 1)
					}
					searchPaths = append(searchPaths, path)
				}
			}

			selectedAction := config.TmuxOpenEditor
			if len(m.tmuxDefaultActions) > 0 && m.tmuxActionIdx < len(m.tmuxDefaultActions) {
				selectedAction = m.tmuxDefaultActions[m.tmuxActionIdx]
			}

			m.completed = &config.Config{
				Editor:                m.editor.Value(),
				SearchPaths:           searchPaths,
				TmuxDefaultOpenAction: string(selectedAction),
			}

			return m, tea.Quit
		case "shift+tab":
			if m.step == 1 {
				m.step = 0
				m.editor.Focus()
				m.paths.Blur()
				return m, nil
			}
			if m.step == 2 {
				m.step = 1
				m.paths.Focus()
				return m, nil
			}
		case "up":
			if m.step == 2 && m.tmuxActionIdx > 0 {
				m.tmuxActionIdx--
			}
			return m, nil

		case "down":
			if m.step == 2 && m.tmuxActionIdx < len(m.tmuxDefaultActions)-1 {
				m.tmuxActionIdx++
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
	}

	if m.step == 0 {
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
		return m, cmd
	}
	if m.step == 1 {
		var cmd tea.Cmd
		m.paths, cmd = m.paths.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m SetupModel) View() string {
	if m.step == 0 {
		return m.editorView()
	}
	if m.step == 1 {
		return m.pathsView()
	}
	return m.tmuxActionView()
}

func (m SetupModel) editorView() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))

	title := headerStyle.Render("🎯 Git Fuzzy Setup")
	subtitle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Step 1 of 3: Editor")

	inputStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 1).Width(50)

	input := inputStyle.Render(m.editor.View())

	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Padding(1, 0)

	footer := footerStyle.Render("Enter: next | Ctrl+C: cancel")

	return fmt.Sprintf("%s\n\n%s\n\nWhat's your preferred editor?\n%s\n\n%s", title, subtitle, input, footer)
}

func (m SetupModel) pathsView() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))

	title := headerStyle.Render("🎯 Git Fuzzy Setup")
	subtitle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Step 2 of 3: Search Paths")

	inputStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 1).Width(50)

	input := inputStyle.Render(m.paths.View())

	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Padding(1, 0)

	footer := footerStyle.Render("Enter: save | Shift+Tab: back | Ctrl+C: cancel")

	return fmt.Sprintf("%s\n%s\n\nEnter directories (comma-separated):\n%s\n\n%s", title, subtitle, input, footer)
}

func (m SetupModel) tmuxActionView() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	title := headerStyle.Render("🎯 Git Fuzzy Setup")
	subtitle := mutedStyle.Render("Step 3 of 3: Tmux Enter Action")

	var lines []string
	for i, action := range m.tmuxDefaultActions {
		label := tmuxActionLabel(action)
		if i == m.tmuxActionIdx {
			lines = append(lines, selectedStyle.Render("▶ "+label))
		} else {
			lines = append(lines, "  "+label)
		}
	}

	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Padding(1, 0)
	footer := footerStyle.Render("↑/↓: choose | Enter: save | Shift+Tab: back | Ctrl+C: cancel")

	return fmt.Sprintf("%s\n%s\n\nDefault action for Enter while inside tmux:\n%s\n\n%s", title, subtitle, strings.Join(lines, "\n"), footer)
}

func tmuxActionLabel(action config.TmuxOpenAction) string {
	switch action {
	case config.TmuxOpenEditor:
		return "Current editor"
	case config.TmuxOpenWindow:
		return "Tmux window"
	case config.TmuxOpenVerticalPane:
		return "Tmux vertical pane"
	case config.TmuxOpenHorizontalPane:
		return "Tmux horizontal pane"
	case config.TmuxOpenSession:
		return "Tmux session"
	default:
		return "Current editor"
	}
}
