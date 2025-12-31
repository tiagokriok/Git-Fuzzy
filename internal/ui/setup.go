package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tiagokriok/gf/internal/config"
)

type SetupModel struct {
	step      int
	editor    textinput.Model
	paths     textinput.Model
	completed *config.Config
	err       error
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
		step:   0,
		editor: editorInput,
		paths:  pathsInput,
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
			} else {
				if m.paths.Value() == "" {
					m.err = fmt.Errorf("paths cannot be empty")
					return m, tea.Quit
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

				m.completed = &config.Config{
					Editor:      m.editor.Value(),
					SearchPaths: searchPaths,
				}

				return m, tea.Quit
			}
		case "shift+tab":
			if m.step == 1 {
				m.step = 0
				m.editor.Focus()
				m.paths.Blur()
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
	}

	if m.step == 0 {
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
		return m, cmd
	} else {
		var cmd tea.Cmd
		m.paths, cmd = m.paths.Update(msg)
		return m, cmd
	}
}

func (m SetupModel) View() string {
	if m.step == 0 {
		return m.editorView()
	} else {
		return m.pathsView()
	}
}

func (m SetupModel) editorView() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))

	title := headerStyle.Render("🎯 Git Finder Setup")
	subtitle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Step 1 of 2: Editor")

	inputStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 1).Width(50)

	input := inputStyle.Render(m.editor.View())

	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Padding(1, 0)

	footer := footerStyle.Render("Enter: next | Ctrl+C: cancel")

	return fmt.Sprintf("%s\n\n%s\n\nWhat's your preferred editor?\n%s\n\n%s", title, subtitle, input, footer)
}

func (m SetupModel) pathsView() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))

	title := headerStyle.Render("🎯 Git Finder Setup")
	subtitle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Step 2 of 2: Search Paths")

	inputStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 1).Width(50)

	input := inputStyle.Render(m.paths.View())

	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Padding(1, 0)

	footer := footerStyle.Render("Enter: save | Shift+Tab: back | Ctrl+C: cancel")

	return fmt.Sprintf("%s\n%s\n\nEnter directories (comma-separated):\n%s\n\n%s", title, subtitle, input, footer)
}
