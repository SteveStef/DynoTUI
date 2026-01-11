package main

import (
	"github.com/charmbracelet/lipgloss"
)

// --- Styles ---

var (
	// Theme Palette
	// Dark background compatible colors
	primary    = lipgloss.Color("#7D56F4") // Purple/Indigo
	secondary  = lipgloss.Color("#04B575") // Teal/Green
	alert      = lipgloss.Color("#FF5F87") // Red/Pink
	textLight  = lipgloss.Color("#E4E4E4") // Off-white
	textDim    = lipgloss.Color("#626262") // Gray
	bgHighlight = lipgloss.Color("#3C3836") // Dark Gray for selection

	// Mapping to existing logic
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	accent    = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#04B575"}
	warning   = lipgloss.AdaptiveColor{Light: "#F25D94", Dark: "#FF5F87"}

	// Global Headers
	headerStyle = lipgloss.NewStyle().
		Foreground(textLight).
		Background(primary).
		Padding(0, 1).
		Bold(true)

	// List Pane
	listHeaderStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(subtle).
		Foreground(secondary).
		Bold(true).
		PaddingLeft(1)

	listItemStyle = lipgloss.NewStyle().
		PaddingLeft(1).
		Foreground(lipgloss.Color("252")) // Standard text

	listSelectedStyle = lipgloss.NewStyle().
		PaddingLeft(1).
		Foreground(lipgloss.Color("#FFF")).
		Background(primary).
		Bold(true).
		BorderLeft(false) // Removed border in favor of background

	// Item View Styles
	itemHeaderStyle = lipgloss.NewStyle().
		Foreground(secondary).
		Bold(true).
		Padding(0, 1).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(subtle)

	itemRowStyle = lipgloss.NewStyle().
		Padding(0, 1)

	tableRowStyle = lipgloss.NewStyle().
		Padding(0, 1)

	tableSelectedRowStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(lipgloss.Color("#FFF")).
		Background(primary)

	// Detail Pane
	detailStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(subtle).
		Padding(0, 1)
	
	labelStyle = lipgloss.NewStyle().Foreground(textDim).Width(12)
	valueStyle = lipgloss.NewStyle().Foreground(textLight)

	// Command Bar
	inputStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primary).
		Padding(0, 1)
	
	placeholderStyle = lipgloss.NewStyle().Foreground(textDim)

	// Dialog
	dialogBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(alert).
		Padding(1, 2).
		BorderTop(true).
		BorderLeft(true).
		BorderRight(true).
		BorderBottom(true)

	// Status Bar
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
)
