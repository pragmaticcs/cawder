package styles

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

var (
	gruvBg     = lipgloss.AdaptiveColor{Light: "#fbf1c7", Dark: "#282828"}
	gruvBg1    = lipgloss.AdaptiveColor{Light: "#ebdbb2", Dark: "#3c3836"}
	gruvFg     = lipgloss.AdaptiveColor{Light: "#3c3836", Dark: "#ebdbb2"}
	gruvGray   = lipgloss.AdaptiveColor{Light: "#928374", Dark: "#928374"}
	gruvRed    = lipgloss.AdaptiveColor{Light: "#cc241d", Dark: "#fb4934"}
	gruvGreen  = lipgloss.AdaptiveColor{Light: "#98971a", Dark: "#b8bb26"}
	gruvYellow = lipgloss.AdaptiveColor{Light: "#d79921", Dark: "#fabd2f"}
	gruvBlue   = lipgloss.AdaptiveColor{Light: "#458588", Dark: "#83a598"}
	gruvPurple = lipgloss.AdaptiveColor{Light: "#b16286", Dark: "#d3869b"}
	gruvAqua   = lipgloss.AdaptiveColor{Light: "#689d6a", Dark: "#8ec07c"}
	gruvOrange = lipgloss.AdaptiveColor{Light: "#d65d0e", Dark: "#fe8019"}
)

var (
	ColorUser   = gruvBlue
	ColorAgent  = gruvGreen
	ColorMuted  = gruvGray
	ColorError  = gruvRed
	ColorAccent = gruvYellow
	ColorTool   = gruvAqua
	ColorWarn   = gruvOrange
)

var (
	AppStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Foreground(gruvFg)

	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(gruvBg).
			Background(ColorAccent).
			Padding(0, 1).
			MarginBottom(1)

	InputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(gruvBlue).
			Foreground(gruvFg).
			Padding(0, 1)

	StatusStyle = lipgloss.NewStyle().
			Faint(true).
			Foreground(ColorMuted)

	FooterStyle = lipgloss.NewStyle().
			MarginTop(1)

	ErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorError).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(ColorError).
			PaddingLeft(1)

	ToolCallStyle = lipgloss.NewStyle().
			Foreground(ColorTool).
			Bold(true)

	ToolCallPendingStyle = lipgloss.NewStyle().
				Foreground(ColorTool).
				Faint(true)

	ToolCallErrorStyle = lipgloss.NewStyle().
				Foreground(ColorError).
				Bold(true)

	ToolResultStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(ColorTool).
			PaddingLeft(1).
			MarginBottom(1)

	ToolErrorResultStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(ColorError).
				PaddingLeft(1).
				MarginBottom(1)

	WarnStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWarn)

	SpinnerStyle = lipgloss.NewStyle().
			Foreground(ColorAccent)
)

var (
	UserResponseStyle = lipgloss.NewStyle().
				Bold(true).
				MarginBottom(1)

	AgentResponseStyle = lipgloss.NewStyle().
				MarginBottom(1)

	AgentReasoningStyle = lipgloss.NewStyle().
				Italic(true).
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(gruvBg1).
				PaddingLeft(1).
				MarginBottom(1)
)

var ListStyles = list.Styles{
	TitleBar: lipgloss.NewStyle().
		Padding(0, 0, 1, 0),

	Title: lipgloss.NewStyle().
		Bold(true).
		Foreground(gruvBg).
		Background(ColorAccent).
		Padding(0, 1),

	FilterPrompt: lipgloss.NewStyle().
		Foreground(ColorAccent),

	FilterCursor: lipgloss.NewStyle().
		Foreground(ColorAccent),

	DefaultFilterCharacterMatch: lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true),

	StatusBar: lipgloss.NewStyle().
		Foreground(ColorMuted),

	StatusEmpty: lipgloss.NewStyle().
		Foreground(ColorMuted),

	StatusBarActiveFilter: lipgloss.NewStyle().
		Foreground(gruvFg),

	StatusBarFilterCount: lipgloss.NewStyle().
		Foreground(ColorMuted),

	NoItems: lipgloss.NewStyle().
		Foreground(ColorMuted).
		Italic(true),

	PaginationStyle: lipgloss.NewStyle(),

	HelpStyle: lipgloss.NewStyle().
		Padding(1, 0, 0, 0),

	ActivePaginationDot: lipgloss.NewStyle().
		Foreground(ColorAccent),

	InactivePaginationDot: lipgloss.NewStyle().
		Foreground(ColorMuted),

	ArabicPagination: lipgloss.NewStyle().
		Foreground(ColorMuted),

	DividerDot: lipgloss.NewStyle().
		Foreground(ColorMuted),
}

var ListItemStyles = list.DefaultItemStyles{
	NormalTitle: lipgloss.NewStyle().
		Foreground(gruvFg).
		Padding(0, 0, 0, 2),

	NormalDesc: lipgloss.NewStyle().
		Foreground(ColorMuted).
		Padding(0, 0, 0, 2),

	SelectedTitle: lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(ColorUser).
		Foreground(ColorUser).
		Bold(true).
		Padding(0, 0, 0, 1),

	SelectedDesc: lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(ColorUser).
		Foreground(ColorMuted).
		Padding(0, 0, 0, 1),

	DimmedTitle: lipgloss.NewStyle().
		Foreground(ColorMuted).
		Faint(true).
		Padding(0, 0, 0, 2),

	DimmedDesc: lipgloss.NewStyle().
		Foreground(ColorMuted).
		Faint(true).
		Padding(0, 0, 0, 2),

	FilterMatch: lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true),
}

var ListHelpStyles = help.Styles{
	Ellipsis:       lipgloss.NewStyle().Foreground(ColorMuted),
	ShortKey:       lipgloss.NewStyle().Foreground(ColorAccent),
	ShortDesc:      lipgloss.NewStyle().Foreground(ColorMuted),
	ShortSeparator: lipgloss.NewStyle().Foreground(ColorMuted),
	FullKey:        lipgloss.NewStyle().Foreground(ColorAccent),
	FullDesc:       lipgloss.NewStyle().Foreground(ColorMuted),
	FullSeparator:  lipgloss.NewStyle().Foreground(ColorMuted),
}
