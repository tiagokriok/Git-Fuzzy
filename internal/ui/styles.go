package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/tiagokriok/Git-Fuzzy/internal/theme"
)

type Styles struct {
	Title        lipgloss.Style
	Subtitle     lipgloss.Style
	Muted        lipgloss.Style
	MutedItalic  lipgloss.Style
	Selected     lipgloss.Style
	Border       lipgloss.Color
	Accent       lipgloss.Color
	Foreground   lipgloss.Color
	Success      lipgloss.Color
	Warning      lipgloss.Color
	Danger       lipgloss.Color
	Info         lipgloss.Color
	Branch       lipgloss.Color
	Stash        lipgloss.Color
	Renamed      lipgloss.Color
	Copied       lipgloss.Color
	Untracked    lipgloss.Color
	SearchBox    lipgloss.Style
	Panel        lipgloss.Style
	Error        lipgloss.Style
	Footer       lipgloss.Style
	FooterPadded lipgloss.Style
}

func NewStyles(appTheme theme.Theme) Styles {
	palette := theme.Default().Palette
	if appTheme.Palette != (theme.Palette{}) {
		palette = appTheme.Palette
	}
	accent := lipgloss.Color(palette.Accent)
	border := lipgloss.Color(palette.Border)
	muted := lipgloss.Color(palette.Muted)
	danger := lipgloss.Color(palette.Danger)
	return Styles{
		Title:        lipgloss.NewStyle().Bold(true).Foreground(accent),
		Subtitle:     lipgloss.NewStyle().Foreground(muted),
		Muted:        lipgloss.NewStyle().Foreground(muted),
		MutedItalic:  lipgloss.NewStyle().Foreground(muted).Italic(true),
		Selected:     lipgloss.NewStyle().Foreground(lipgloss.Color(palette.Selection)).Bold(true),
		Border:       border,
		Accent:       accent,
		Foreground:   lipgloss.Color(palette.Foreground),
		Success:      lipgloss.Color(palette.Success),
		Warning:      lipgloss.Color(palette.Warning),
		Danger:       danger,
		Info:         lipgloss.Color(palette.Info),
		Branch:       lipgloss.Color(palette.Branch),
		Stash:        lipgloss.Color(palette.Stash),
		Renamed:      lipgloss.Color(palette.Renamed),
		Copied:       lipgloss.Color(palette.Copied),
		Untracked:    lipgloss.Color(palette.Untracked),
		SearchBox:    lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent).Padding(0, 1).Align(lipgloss.Left),
		Panel:        lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(border).Padding(1),
		Error:        lipgloss.NewStyle().Foreground(danger),
		Footer:       lipgloss.NewStyle().Foreground(muted).Align(lipgloss.Center),
		FooterPadded: lipgloss.NewStyle().Foreground(muted).Padding(1, 0),
	}
}

func themeWarningText(warning *theme.Warning) string {
	if warning == nil {
		return ""
	}
	return warning.Message
}
