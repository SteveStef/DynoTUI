package main

import (
	"github.com/charmbracelet/lipgloss"
)

// --- Themes ---

type Theme struct {
	Primary     lipgloss.Color
	Secondary   lipgloss.Color
	Alert       lipgloss.Color
	TextLight   lipgloss.Color
	TextDim     lipgloss.Color
	BgHighlight lipgloss.Color
	Subtle      lipgloss.AdaptiveColor
	Highlight   lipgloss.AdaptiveColor
	Accent      lipgloss.AdaptiveColor
	Warning     lipgloss.AdaptiveColor
}

var (
	Themes = map[string]Theme{
		"Dark": {
			Primary:     lipgloss.Color("#7D56F4"), // Purple
			Secondary:   lipgloss.Color("#04B575"), // Teal
			Alert:       lipgloss.Color("#FF5F87"), // Pink
			TextLight:   lipgloss.Color("#E4E4E4"),
			TextDim:     lipgloss.Color("#626262"),
			BgHighlight: lipgloss.Color("#3C3836"),
			Subtle:      lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"},
			Highlight:   lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"},
			Accent:      lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#04B575"},
			Warning:     lipgloss.AdaptiveColor{Light: "#F25D94", Dark: "#FF5F87"},
		},
		"Dracula": {
			Primary:     lipgloss.Color("#BD93F9"), // Purple
			Secondary:   lipgloss.Color("#50FA7B"), // Green
			Alert:       lipgloss.Color("#FF5555"), // Red
			TextLight:   lipgloss.Color("#F8F8F2"),
			TextDim:     lipgloss.Color("#6272A4"),
			BgHighlight: lipgloss.Color("#44475A"),
			Subtle:      lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#44475A"},
			Highlight:   lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#BD93F9"},
			Accent:      lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#50FA7B"},
			Warning:     lipgloss.AdaptiveColor{Light: "#F25D94", Dark: "#FF5555"},
		},
		"Blue": {
			Primary:     lipgloss.Color("#3B8ED0"), // Blue
			Secondary:   lipgloss.Color("#E0DEF4"), // Light
			Alert:       lipgloss.Color("#EB6F92"), // Rose
			TextLight:   lipgloss.Color("#E0DEF4"),
			TextDim:     lipgloss.Color("#908CAA"),
			BgHighlight: lipgloss.Color("#26233A"),
			Subtle:      lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#26233A"},
			Highlight:   lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#3B8ED0"},
			Accent:      lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#9CCFD8"},
			Warning:     lipgloss.AdaptiveColor{Light: "#F25D94", Dark: "#EB6F92"},
		},
		"Monokai": {
			Primary:     lipgloss.Color("#A6E22E"), // Green
			Secondary:   lipgloss.Color("#66D9EF"), // Cyan
			Alert:       lipgloss.Color("#F92672"), // Pink
			TextLight:   lipgloss.Color("#F8F8F2"),
			TextDim:     lipgloss.Color("#75715E"),
			BgHighlight: lipgloss.Color("#272822"),
			Subtle:      lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#272822"},
			Highlight:   lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#A6E22E"},
			Accent:      lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#66D9EF"},
			Warning:     lipgloss.AdaptiveColor{Light: "#F25D94", Dark: "#F92672"},
		},
		"Synthwave": {
			Primary:     lipgloss.Color("#FF71CE"), // Neon Pink
			Secondary:   lipgloss.Color("#01CDFE"), // Neon Cyan
			Alert:       lipgloss.Color("#FF0055"), // Red
			TextLight:   lipgloss.Color("#FFF"),
			TextDim:     lipgloss.Color("#B967FF"), // Purple Dim
			BgHighlight: lipgloss.Color("#2B213A"),
			Subtle:      lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#2B213A"},
			Highlight:   lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#FF71CE"},
			Accent:      lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#01CDFE"},
			Warning:     lipgloss.AdaptiveColor{Light: "#F25D94", Dark: "#FF0055"},
		},
		"Solarized Light": {
			Primary:     lipgloss.Color("#268BD2"), // Blue
			Secondary:   lipgloss.Color("#859900"), // Green
			Alert:       lipgloss.Color("#DC322F"), // Red
			TextLight:   lipgloss.Color("#586E75"), // Dark Gray (Text)
			TextDim:     lipgloss.Color("#93A1A1"), // Light Gray
			BgHighlight: lipgloss.Color("#EEE8D5"), // Cream
			Subtle:      lipgloss.AdaptiveColor{Light: "#EEE8D5", Dark: "#073642"},
			Highlight:   lipgloss.AdaptiveColor{Light: "#268BD2", Dark: "#268BD2"},
			Accent:      lipgloss.AdaptiveColor{Light: "#859900", Dark: "#859900"},
			Warning:     lipgloss.AdaptiveColor{Light: "#DC322F", Dark: "#DC322F"},
		},
	}

	currentThemeName = "Dark"
	
	// Global Colors (Updated by SetTheme)
	primary, secondary, alert, textLight, textDim, bgHighlight lipgloss.Color
	subtle, highlight, accent, warning lipgloss.AdaptiveColor

	// Global Styles (Updated by SetTheme)
	headerStyle, listHeaderStyle, listItemStyle, listSelectedStyle lipgloss.Style
	itemHeaderStyle, itemRowStyle, tableRowStyle, tableSelectedRowStyle lipgloss.Style
	detailStyle, labelStyle, valueStyle, inputStyle, placeholderStyle lipgloss.Style
	dialogBoxStyle, statusBarStyle, statusKeyStyle, statusValStyle lipgloss.Style
)

func init() {
	cfg, err := LoadConfig()
	if err != nil {
		// Fallback to default if load fails
		SetTheme("Dark")
	} else {
		// Verify theme exists
		if _, ok := Themes[cfg.Theme]; ok {
			SetTheme(cfg.Theme)
		} else {
			SetTheme("Dark")
		}
	}
}

func SetTheme(name string) {
	t, ok := Themes[name]
	if !ok {
		return
	}
	currentThemeName = name

	// Update Colors
	primary = t.Primary
	secondary = t.Secondary
	alert = t.Alert
	textLight = t.TextLight
	textDim = t.TextDim
	bgHighlight = t.BgHighlight
	subtle = t.Subtle
	highlight = t.Highlight
	accent = t.Accent
	warning = t.Warning

	// Rebuild Styles
	headerStyle = lipgloss.NewStyle().
		Foreground(textLight).
		Background(primary).
		Padding(0, 1).
		Bold(true)

	listHeaderStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(subtle).
		Foreground(secondary).
		Bold(true).
		PaddingLeft(1)

	listItemStyle = lipgloss.NewStyle().
		PaddingLeft(1).
		Foreground(lipgloss.Color("252"))

	listSelectedStyle = lipgloss.NewStyle().
		PaddingLeft(1).
		Foreground(lipgloss.Color("#FFF")).
		Background(primary).
		Bold(true).
		BorderLeft(false)

	itemHeaderStyle = lipgloss.NewStyle().
		Foreground(secondary).
		Bold(true).
		Padding(0, 1).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(subtle)

	itemRowStyle = lipgloss.NewStyle().Padding(0, 1)

	tableRowStyle = lipgloss.NewStyle().Padding(0, 1)

	tableSelectedRowStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(lipgloss.Color("#FFF")).
		Background(primary)

	detailStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(subtle).
		Padding(0, 1)

	labelStyle = lipgloss.NewStyle().Foreground(textDim).Width(12)
	valueStyle = lipgloss.NewStyle().Foreground(textLight)

	inputStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primary).
		Padding(0, 1)

	placeholderStyle = lipgloss.NewStyle().Foreground(textDim)

	dialogBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(alert).
		Padding(1, 2).
		BorderTop(true).
		BorderLeft(true).
		BorderRight(true).
		BorderBottom(true)

	statusBarStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFF")).
		Background(lipgloss.AdaptiveColor{Light: "#355C7D", Dark: "#2A2A2A"})

	statusKeyStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFF")).
		Background(primary).
		Padding(0, 1)

	statusValStyle = lipgloss.NewStyle().
		Foreground(textLight).
		Background(lipgloss.AdaptiveColor{Light: "#355C7D", Dark: "#2A2A2A"}).
		Padding(0, 1)
}

func NextTheme() string {
	order := []string{"Dark", "Dracula", "Blue", "Monokai", "Synthwave", "Solarized Light"}
	next := ""
	for i, name := range order {
		if name == currentThemeName {
			if i+1 < len(order) {
				next = order[i+1]
			} else {
				next = order[0]
			}
			break
		}
	}
	SetTheme(next)

	// Save the new preference
	_ = SaveConfig(Config{Theme: next})

	return next
}
