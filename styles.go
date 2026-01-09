package main

import (
	"github.com/charmbracelet/lipgloss"
)

// --- Styles ---

var (
	// Theme
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	accent    = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	warning   = lipgloss.AdaptiveColor{Light: "#F25D94", Dark: "#F25D94"}

	// Global Headers
	headerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFDF5")).
		Background(highlight).
		Padding(0, 1).
		Bold(true)

	// List Pane
	listHeaderStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(subtle).
		Foreground(highlight).
		Bold(true)

	listItemStyle = lipgloss.NewStyle().PaddingLeft(1).Foreground(lipgloss.Color("241"))
	listSelectedStyle = lipgloss.NewStyle().PaddingLeft(1).Foreground(highlight).Bold(true).
		BorderLeft(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(highlight)

	// Item View Styles
	itemHeaderStyle = lipgloss.NewStyle().
		Foreground(accent).
		Bold(true).
		Padding(0, 1).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(subtle)

	itemRowStyle = lipgloss.NewStyle().
		Padding(0, 1)

	// Detail Pane
	detailStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(subtle).
		Padding(0, 1)
	
	labelStyle = lipgloss.NewStyle().Foreground(subtle).Width(12)
	valueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	// Command Bar
	inputStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(highlight).
		Padding(0, 1)

	// Dialog
	dialogBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(warning).
		Padding(1, 0).
		BorderTop(true).
		BorderLeft(true).
		BorderRight(true).
		BorderBottom(true)
)
