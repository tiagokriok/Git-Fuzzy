package ui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
	"github.com/tiagokriok/Git-Fuzzy/internal/config"
	"github.com/tiagokriok/Git-Fuzzy/internal/git"
	"github.com/tiagokriok/Git-Fuzzy/internal/platform"
	"github.com/tiagokriok/Git-Fuzzy/internal/scanner"
	"github.com/tiagokriok/Git-Fuzzy/internal/tmux"
)

var selectedRepository *scanner.Repository

const (
	maxHeight       = 10
	boxPadding      = 2
	searchBoxHeight = 3
	footerHeight    = 3
	searchBoxWidth  = 50
)

type uiMode int

const (
	modeRepositoryList uiMode = iota
	modeSessionPrompt
)

// Message types for async operations
type gitStatusFetchMsg struct {
	data     *git.StatusData
	err      error
	repoPath string
}

type debounceTickMsg struct {
	repoPath string
}

type Model struct {
	repositories       []scanner.Repository
	filtered           []scanner.Repository
	searchInput        string
	selectedIdx        int
	scrollOffset       int
	width              int
	height             int
	err                error
	gitStatusData      *git.StatusData
	gitStatusScroll    int
	gitStatusLoading   bool
	gitStatusError     error
	config             *config.Config
	mode               uiMode
	statusMessage      string
	sessionNameInput   string
	sessionPromptError string
}

func NewModel(repos []scanner.Repository, cfg *config.Config) Model {
	return Model{
		repositories: repos,
		filtered:     repos,
		selectedIdx:  0,
		config:       cfg,
		mode:         modeRepositoryList,
	}
}

func (m Model) Init() tea.Cmd {
	// Fetch git status for first repository (no debounce delay)
	if len(m.repositories) > 0 {
		return m.fetchGitStatusAsync(m.repositories[0].Path)
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case debounceTickMsg:
		// Only fetch if still on the same repo
		if len(m.filtered) > 0 && m.selectedIdx < len(m.filtered) {
			selected := m.filtered[m.selectedIdx]
			if selected.Path == msg.repoPath {
				m.gitStatusLoading = true
				return m, m.fetchGitStatusAsync(selected.Path)
			}
		}
		return m, nil
	case gitStatusFetchMsg:
		m.gitStatusLoading = false
		if msg.err != nil {
			m.gitStatusError = msg.err
			m.gitStatusData = nil
		} else {
			m.gitStatusData = msg.data
			m.gitStatusError = nil
			m.gitStatusScroll = 0
		}
		return m, nil
	}
	return m, nil
}

func (m Model) View() string {
	if m.mode == modeSessionPrompt {
		return m.renderSessionPrompt()
	}

	// Calculate panel widths (60/40 split)
	// Each panel has: border (2) + padding (2) = 4 extra chars
	// We need to account for this "chrome" when calculating content widths
	panelChrome := 4                     // border (2) + padding (2) per panel
	totalChrome := (panelChrome * 2) + 1 // both panels + 1 char gap between them
	totalContentWidth := m.width - totalChrome

	// Ensure minimum usable width
	if totalContentWidth < 40 {
		totalContentWidth = 40
	}

	leftPanelWidth := int(float64(totalContentWidth) * 0.55)
	rightPanelWidth := totalContentWidth - leftPanelWidth

	// Render both panels
	leftPanel := m.renderLeftPanel(leftPanelWidth)
	rightPanel := m.renderRightPanel(rightPanelWidth)

	// Join horizontally
	splitView := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	// Add footer
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, splitView, footer)
}

func (m Model) renderLeftPanel(width int) string {
	searchBoxWidth := min(width-4, 50)
	searchBoxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("205")).Padding(0, 1).Width(searchBoxWidth).Align(lipgloss.Left)

	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)

	availableHeight := max(m.height-footerHeight-4, 3)

	searchLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Search:")
	searchInput := m.searchInput
	searchBox := searchBoxStyle.Render(searchInput)

	var reposList string
	if len(m.filtered) == 0 {
		emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
		reposList = emptyStyle.Render("No repositories found")
	} else {
		var lines []string

		itemsToShow := min(len(m.filtered), maxHeight)

		if m.selectedIdx < m.scrollOffset {
			m.scrollOffset = m.selectedIdx
		}
		if m.selectedIdx >= m.scrollOffset+itemsToShow {
			m.scrollOffset = m.selectedIdx - itemsToShow + 1
		}

		for i := 0; i < itemsToShow && i < availableHeight; i++ {
			repoIdx := m.scrollOffset + i
			if repoIdx >= len(m.filtered) {
				break
			}
			repo := m.filtered[repoIdx]
			displayPath := formatRepoPath(repo.Path)
			line := fmt.Sprintf("%s (%s)", repo.Name, displayPath)

			if repoIdx == m.selectedIdx {
				lines = append(lines, selectedStyle.Render("▶ "+line))
			} else {
				lines = append(lines, "  "+line)
			}
		}

		reposList = strings.Join(lines, "\n")
	}
	paginationInfo := m.getPaginationInfo()
	paginationStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
	pagination := paginationStyle.Render(paginationInfo)

	content := lipgloss.JoinVertical(lipgloss.Left, searchLabel, searchBox, "", reposList, pagination)

	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1).
		Width(width).
		Height(m.height - footerHeight - 2)

	return panelStyle.Render(content)
}

func (m Model) renderRightPanel(width int) string {
	// Panel title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Padding(0, 1)
	title := titleStyle.Render("📊 Git Status")

	var content string

	if len(m.filtered) == 0 {
		// No repositories
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true).
			Padding(2, 1)
		content = emptyStyle.Render("No repository selected")

	} else if m.gitStatusLoading {
		// Loading state
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true).
			Padding(2, 1)
		content = loadingStyle.Render("Loading git status...")

	} else if m.gitStatusError != nil {
		// Error state
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Padding(2, 1)
		content = errorStyle.Render(fmt.Sprintf("⚠ Error:\n\n%s", m.gitStatusError.Error()))

	} else if m.gitStatusData != nil {
		// Render git status content
		content = m.renderGitStatusContent(width)
	} else {
		// Initial state (no data fetched yet)
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true).
			Padding(2, 1)
		content = emptyStyle.Render("Select a repository to view status")
	}

	// Wrap in border
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1).
		Width(width).
		Height(m.height - footerHeight - 2)

	return panelStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title, "", content))
}

func (m Model) renderGitStatusContent(width int) string {
	data := m.gitStatusData

	// Branch header
	branchStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("46")).
		Bold(true).
		Padding(0, 1)
	branchHeader := branchStyle.Render(fmt.Sprintf("🌿 %s", data.CurrentBranch))

	// Tracking branch
	var trackingLine string
	if data.TrackingBranch != "" {
		trackingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(0, 1)
		trackingLine = trackingStyle.Render(fmt.Sprintf("└─ tracking: %s", data.TrackingBranch))
	}

	// Stats section
	statsSection := m.renderStatsSection(data)

	// Files section with scrolling
	filesSection := m.renderFilesSection(data, width)

	// Assemble
	return lipgloss.JoinVertical(
		lipgloss.Left,
		branchHeader,
		trackingLine,
		"",
		statsSection,
		"",
		filesSection,
	)
}

func (m Model) renderStatsSection(data *git.StatusData) string {
	statsStyle := lipgloss.NewStyle().Padding(0, 1).Foreground(lipgloss.Color("250"))

	var statLines []string

	// Ahead/Behind
	if data.AheadCount > 0 || data.BehindCount > 0 {
		aheadBehind := ""
		if data.AheadCount > 0 {
			aheadStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
			aheadBehind += aheadStyle.Render(fmt.Sprintf("⬆ %d", data.AheadCount))
		}
		if data.BehindCount > 0 {
			if aheadBehind != "" {
				aheadBehind += "  "
			}
			behindStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
			aheadBehind += behindStyle.Render(fmt.Sprintf("⬇ %d", data.BehindCount))
		}
		statLines = append(statLines, statsStyle.Render(aheadBehind))
	}

	// File change summary
	changeCount := data.ModifiedCount + data.AddedCount + data.DeletedCount + data.UntrackedCount

	if changeCount > 0 {
		summary := fmt.Sprintf("📊 %d file%s changed", changeCount, m.pluralize(changeCount))

		breakdown := ""
		if data.AddedCount > 0 {
			breakdown += lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render(fmt.Sprintf("+%d", data.AddedCount)) + " "
		}
		if data.ModifiedCount > 0 {
			breakdown += lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render(fmt.Sprintf("~%d", data.ModifiedCount)) + " "
		}
		if data.DeletedCount > 0 {
			breakdown += lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("-%d", data.DeletedCount))
		}
		if data.UntrackedCount > 0 {
			if breakdown != "" {
				breakdown += " "
			}
			breakdown += lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Render(fmt.Sprintf("?%d", data.UntrackedCount))
		}
		if breakdown != "" {
			summary += " (" + strings.Trim(breakdown, " ") + ")"
		}

		statLines = append(statLines, statsStyle.Render(summary))
	}

	return strings.Join(statLines, "\n")
}

func (m Model) renderFilesSection(data *git.StatusData, width int) string {
	accentColor := lipgloss.Color("205")

	filesTitle := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true).
		Padding(0, 1).
		Render("📝 Files")

	var fileLines []string
	visibleHeight := min(m.height-18, 15)
	maxFiles := min(len(data.Files)-m.gitStatusScroll, visibleHeight)

	// Calculate max filename width: panel width - borders - padding - prefix (symbol + status)
	// Prefix is roughly: emoji(2) + space(1) + status(2) + spaces(2) + padding(2) = ~9 chars
	maxFilenameWidth := width - 12
	if maxFilenameWidth < 20 {
		maxFilenameWidth = 20
	}

	if len(data.Files) == 0 {
		cleanStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true).
			Padding(0, 1)
		fileLines = append(fileLines, cleanStyle.Render("✓ Working tree clean"))
	} else {
		statusColors := map[string]lipgloss.Color{
			"M":  lipgloss.Color("220"), // Yellow for modified
			"A":  lipgloss.Color("46"),  // Green for added
			"D":  lipgloss.Color("196"), // Red for deleted
			"R":  lipgloss.Color("171"), // Magenta for renamed
			"C":  lipgloss.Color("51"),  // Cyan for copied
			"??": lipgloss.Color("33"),  // Blue for untracked
		}

		statusSymbols := map[string]string{
			"M":  "✏️ ",
			"A":  "✨",
			"D":  "🗑️ ",
			"R":  "↪️ ",
			"C":  "📋",
			"??": "❓",
		}

		for i := 0; i < maxFiles && i < visibleHeight; i++ {
			file := data.Files[m.gitStatusScroll+i]
			status := file.Status
			color := statusColors[status]
			symbol := statusSymbols[status]
			if symbol == "" {
				symbol = "📄"
			}

			fileStyle := lipgloss.NewStyle().
				Foreground(color).
				Padding(0, 1)

			// Truncate long filenames from the left
			filename := truncatePathLeft(file.Filename, maxFilenameWidth)
			fileLine := fmt.Sprintf("%s %s  %s", symbol, status, filename)
			fileLines = append(fileLines, fileStyle.Render(fileLine))
		}
	}

	filesContent := strings.Join(fileLines, "\n")

	// Scroll indicator
	var scrollIndicator string
	if len(data.Files) > visibleHeight {
		scrollStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
		current := min(m.gitStatusScroll+visibleHeight, len(data.Files))
		scrollIndicator = scrollStyle.Render(
			fmt.Sprintf("(Shift+↑/↓ to scroll: %d-%d of %d)",
				m.gitStatusScroll+1, current, len(data.Files)),
		)
	}

	content := []string{filesTitle, filesContent}
	if scrollIndicator != "" {
		content = append(content, "", scrollIndicator)
	}

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

func (m Model) renderFooter() string {
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Align(lipgloss.Center)
	footerText := "↑/↓: nav repos | Shift+↑/↓: scroll status | Enter: open | Alt+w/v/h/s: tmux | ^O: files | ^T: term | ^B: remote | ^G: refresh | Esc: exit"
	if m.statusMessage != "" {
		footerText = m.statusMessage + " | " + footerText
	}
	return footerStyle.Render(footerText)
}

func (m Model) renderSessionPrompt() string {
	boxWidth := min(max(m.width-8, 40), 80)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	inputStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 1).Width(boxWidth - 4)
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	input := inputStyle.Render(m.sessionNameInput)
	lines := []string{
		titleStyle.Render("New tmux session"),
		"Session name:",
		input,
	}

	if m.sessionPromptError != "" {
		lines = append(lines, errorStyle.Render(m.sessionPromptError))
	}

	lines = append(lines, footerStyle.Render("Enter: create | Esc: cancel | Ctrl+C: exit"))

	panel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(boxWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel)
}

func (m *Model) pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
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

func (m Model) getPaginationInfo() string {
	if len(m.filtered) == 0 {
		return ""
	}

	itemsToShow := min(len(m.filtered), maxHeight)

	if len(m.filtered) <= maxHeight {
		return fmt.Sprintf("(%d results)", len(m.filtered))
	}

	return fmt.Sprintf("Showing %d of %d", itemsToShow, len(m.filtered))
}

func (m Model) scheduleGitStatusFetch() tea.Cmd {
	if len(m.filtered) == 0 {
		return nil
	}

	selected := m.filtered[m.selectedIdx]
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return debounceTickMsg{repoPath: selected.Path}
	})
}

func (m Model) fetchGitStatusAsync(repoPath string) tea.Cmd {
	return func() tea.Msg {
		data, err := git.GetDetailedStatus(repoPath)
		return gitStatusFetchMsg{
			data:     data,
			err:      err,
			repoPath: repoPath,
		}
	}
}

func validateSessionNameInput(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("session name required")
	}
	return nil
}

func (m *Model) startSessionPrompt() {
	m.mode = modeSessionPrompt
	m.sessionNameInput = ""
	m.sessionPromptError = ""
	m.statusMessage = ""
}

func (m *Model) cancelSessionPrompt() {
	m.mode = modeRepositoryList
	m.sessionNameInput = ""
	m.sessionPromptError = ""
}

func (m *Model) selectedRepositoryPath() (string, bool) {
	if len(m.filtered) == 0 || m.selectedIdx >= len(m.filtered) {
		return "", false
	}
	return m.filtered[m.selectedIdx].Path, true
}

func (m *Model) executeTmuxAction(action config.TmuxOpenAction) (tea.Model, tea.Cmd) {
	repoPath, ok := m.selectedRepositoryPath()
	if !ok {
		return m, nil
	}

	if action != config.TmuxOpenEditor && !tmux.IsAvailable() {
		m.statusMessage = "tmux is not available"
		return m, nil
	}

	var err error
	switch action {
	case config.TmuxOpenWindow:
		err = tmux.OpenWindow(repoPath)
	case config.TmuxOpenVerticalPane:
		err = tmux.OpenVerticalPane(repoPath)
	case config.TmuxOpenHorizontalPane:
		err = tmux.OpenHorizontalPane(repoPath)
	case config.TmuxOpenSession:
		m.startSessionPrompt()
		return m, nil
	case config.TmuxOpenEditor:
		selected := m.filtered[m.selectedIdx]
		selectedRepository = &selected
		return m, tea.Quit
	default:
		selected := m.filtered[m.selectedIdx]
		selectedRepository = &selected
		return m, tea.Quit
	}

	if err != nil {
		m.statusMessage = err.Error()
		return m, nil
	}

	selectedRepository = nil
	return m, tea.Quit
}

func (m *Model) submitSessionPrompt() (tea.Model, tea.Cmd) {
	if err := validateSessionNameInput(m.sessionNameInput); err != nil {
		m.sessionPromptError = err.Error()
		return m, nil
	}

	repoPath, ok := m.selectedRepositoryPath()
	if !ok {
		m.cancelSessionPrompt()
		return m, nil
	}

	if err := tmux.OpenSession(strings.TrimSpace(m.sessionNameInput), repoPath); err != nil {
		m.sessionPromptError = err.Error()
		return m, nil
	}

	selectedRepository = nil
	return m, tea.Quit
}

func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.mode == modeSessionPrompt {
		switch msg.String() {
		case "ctrl+c":
			selectedRepository = nil
			return m, tea.Quit
		case "esc":
			m.cancelSessionPrompt()
			return m, nil
		case "enter":
			return m.submitSessionPrompt()
		case "backspace":
			if len(m.sessionNameInput) > 0 {
				m.sessionNameInput = m.sessionNameInput[:len(m.sessionNameInput)-1]
			}
			m.sessionPromptError = ""
			return m, nil
		default:
			m.sessionNameInput += msg.String()
			m.sessionPromptError = ""
			return m, nil
		}
	}

	switch msg.String() {
	case "ctrl+c", "esc":
		selectedRepository = nil
		return m, tea.Quit

	case "alt+w":
		return m.executeTmuxAction(config.TmuxOpenWindow)

	case "alt+v":
		return m.executeTmuxAction(config.TmuxOpenVerticalPane)

	case "alt+h":
		return m.executeTmuxAction(config.TmuxOpenHorizontalPane)

	case "alt+s":
		return m.executeTmuxAction(config.TmuxOpenSession)

	case "ctrl+o": // Open file manager
		if len(m.filtered) > 0 {
			selected := m.filtered[m.selectedIdx]
			m.openFileManager(selected.Path)
		}
		return m, nil

	case "ctrl+t": // Open terminal
		if len(m.filtered) > 0 {
			selected := m.filtered[m.selectedIdx]
			m.openTerminal(selected.Path)
		}
		return m, nil

	case "ctrl+b": // Open in browser
		if len(m.filtered) > 0 {
			selected := m.filtered[m.selectedIdx]
			m.openInBrowser(selected.Path)
		}
		return m, nil

	case "ctrl+g": // Force refresh git status (bypass debounce)
		if len(m.filtered) > 0 {
			selected := m.filtered[m.selectedIdx]
			m.gitStatusLoading = true
			return m, m.fetchGitStatusAsync(selected.Path)
		}
		return m, nil

	case "up", "shift+tab":
		if m.selectedIdx > 0 {
			m.selectedIdx--
			return m, m.scheduleGitStatusFetch()
		}
		return m, nil

	case "down", "tab":
		if m.selectedIdx < len(m.filtered)-1 {
			m.selectedIdx++
			return m, m.scheduleGitStatusFetch()
		}
		return m, nil

	case "shift+up": // Scroll right panel up
		if m.gitStatusData != nil && m.gitStatusScroll > 0 {
			m.gitStatusScroll--
		}
		return m, nil

	case "shift+down": // Scroll right panel down
		if m.gitStatusData != nil && m.gitStatusScroll < len(m.gitStatusData.Files)-1 {
			m.gitStatusScroll++
		}
		return m, nil

	case "enter":
		if len(m.filtered) > 0 {
			if tmux.IsAvailable() {
				return m.executeTmuxAction(m.config.GetTmuxDefaultOpenAction())
			}

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
			m.scrollOffset = 0
			return m, m.scheduleGitStatusFetch()
		}
		return m, nil

	default:
		m.searchInput += msg.String()
		m.updateFiltered()
		m.selectedIdx = 0
		m.scrollOffset = 0
		return m, m.scheduleGitStatusFetch()
	}
}

func formatRepoPath(fullPath string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fullPath
	}

	if strings.HasPrefix(fullPath, homeDir) {
		return strings.TrimPrefix(fullPath, homeDir+"/")
	}
	return fullPath
}

// truncatePathLeft truncates a path from the left if it exceeds maxWidth
// Example: "src/components/dialogs/file.vue" -> "…/dialogs/file.vue"
func truncatePathLeft(path string, maxWidth int) string {
	if len(path) <= maxWidth {
		return path
	}

	// Need at least space for "…/" + some content
	if maxWidth < 5 {
		return path[:maxWidth]
	}

	// Find a good truncation point (prefer path separators)
	targetLen := maxWidth - 1 // -1 for ellipsis
	truncated := path[len(path)-targetLen:]

	// Try to start at a path separator for cleaner output
	if idx := strings.Index(truncated, "/"); idx != -1 && idx < len(truncated)-1 {
		truncated = truncated[idx:]
	}

	return "…" + truncated
}

func (m *Model) openFileManager(repoPath string) {
	cmd := m.config.GetFileManager()
	if cmd == "" {
		return
	}

	// Fire and forget
	go func() {
		exec.Command(cmd, repoPath).Start()
	}()
}

func (m *Model) openTerminal(repoPath string) {
	cmd := m.config.GetTerminal()
	if cmd == "" {
		return
	}

	// Fire and forget
	go func() {
		parts := strings.Fields(cmd)
		if len(parts) > 0 {
			c := exec.Command(parts[0], append(parts[1:], repoPath)...)
			c.Dir = repoPath
			c.Start()
		}
	}()
}

func (m *Model) openInBrowser(repoPath string) {
	remoteURL, err := git.GetRemoteURL(repoPath)
	if err != nil {
		// Silently fail - no remote configured
		return
	}

	httpsURL, err := git.ConvertToHTTPS(remoteURL)
	if err != nil {
		return
	}

	platform.OpenInBrowser(httpsURL)
}

func GetSelectedRepository() *scanner.Repository {
	return selectedRepository
}

func Run(repos []scanner.Repository, cfg *config.Config) (*scanner.Repository, error) {
	selectedRepository = nil

	model := NewModel(repos, cfg)

	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		return nil, fmt.Errorf("TUI Error: %w", err)
	}

	return selectedRepository, nil
}
