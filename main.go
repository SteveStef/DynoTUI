package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Styles ---

var (
	// Theme
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	accent    = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	warning   = lipgloss.AdaptiveColor{Light: "#F25D94", Dark: "#F25D94"}

	// Layout
	mainStyle = lipgloss.NewStyle().Margin(0)

	// Header / Tabs
	activeTabStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#131313")).
		Background(highlight).
		Padding(0, 1)

	inactiveTabStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#D9DCCF")).
		Padding(0, 1)

	awsContextStyle = lipgloss.NewStyle().
		Foreground(warning).
		Align(lipgloss.Right)

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

	// Detail Pane
	detailStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(subtle).
		Padding(0, 1)
	
	detailTitleStyle = lipgloss.NewStyle().
		Foreground(accent).
		Bold(true).
		Underline(true).
		MarginBottom(1)
	
	labelStyle = lipgloss.NewStyle().Foreground(subtle).Width(12)
	valueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	// Command Bar
	inputStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(highlight).
		Padding(0, 1)
)

// --- Model ---

type Table struct {
	Name      string
	PK        string
	SK        string
	Region    string
	ItemCount int
	GSIs      []string
	Status    string
}

type model struct {
	tables    []Table
	cursor    int
	width     int
	height    int
	
	// Navigation
	tabs      []string
	activeTab int
	
	// Input
	input     textinput.Model
	inputMode bool
}

func initialModel() model {

ti := textinput.New()
ti.Placeholder = "Type a command (e.g. 'seed 50 users' or 'scan')"
ti.Focus()
ti.Prompt = "❯ "
ti.CharLimit = 156
ti.Width = 50

return model{
	tables: []Table{
		{"Users", "user_id", "metadata", "us-east-1", 1250, []string{"email-index", "phone-index"}, "ACTIVE"},
		{"Orders", "order_id", "timestamp", "us-east-1", 45000, []string{"customer-date-index"}, "ACTIVE"},
		{"Products", "sku", "category", "us-west-2", 300, []string{}, "ACTIVE"},
		{"Inventory", "warehouse_id", "sku", "us-east-1", 12, []string{"sku-index"}, "UPDATING"},
		{"AuditLogs", "service", "timestamp", "eu-central-1", 890000, []string{}, "ARCHIVING"},
	},
	tabs:      []string{"Tables", "Query", "History", "Settings"},
			activeTab: 0,	input:     ti,
	inputMode: false,
}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.input.Width = msg.Width - 10

	case tea.KeyMsg:
		// Toggle Input Mode
		if msg.String() == "/" && !m.inputMode {
			m.inputMode = true
			m.input.Focus()
			return m, textinput.Blink
		}
		
		if m.inputMode {
			switch msg.String() {
			case "enter":
				m.input.SetValue("") // Clear input on enter (simulated execute)
				m.inputMode = false
				m.input.Blur()
			case "esc":
				m.inputMode = false
				m.input.Blur()
			}
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

		// Navigation Mode
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 { m.cursor-- }
		case "down", "j":
			if m.cursor < len(m.tables)-1 { m.cursor++ }
		case "tab":
			m.activeTab = (m.activeTab + 1) % len(m.tabs)
		case "shift+tab":
			m.activeTab--
			if m.activeTab < 0 { m.activeTab = len(m.tabs) - 1 }
		}
	}

	return m, nil
}

func (m model) View() string {
	if m.width == 0 { return "Initializing..." }

	// --- 1. Header & Tabs ---
	var tabs []string
	for i, t := range m.tabs {
		if i == m.activeTab {
			tabs = append(tabs, activeTabStyle.Render(t))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(t))
		}
	}
	// Add spacer
	tabsStr := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	awsStatus := awsContextStyle.Render("AWS: default (us-east-1) ●")
	
	// Gap between tabs and status
	gapWidth := m.width - lipgloss.Width(tabsStr) - lipgloss.Width(awsStatus) - 2
	if gapWidth < 0 { gapWidth = 0 }
	header := lipgloss.JoinHorizontal(lipgloss.Top, tabsStr, strings.Repeat(" ", gapWidth), awsStatus)
	header = lipgloss.NewStyle().BorderBottom(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(subtle).Width(m.width).Render(header)

	// --- 2. Main Content (Split View) ---
	
	// Left Pane: Table List
	leftWidth := int(float64(m.width) * 0.4)
	var listItems []string
	
	// List Header
	listHeader := listHeaderStyle.Width(leftWidth-2).Render("  NAME")
	listItems = append(listItems, listHeader)

	for i, t := range m.tables {
		str := fmt.Sprintf("  %s", t.Name)
		if m.cursor == i {
			listItems = append(listItems, listSelectedStyle.Width(leftWidth-2).Render(str))
		} else {
			listItems = append(listItems, listItemStyle.Width(leftWidth-2).Render(str))
		}
	}
	leftPane := lipgloss.JoinVertical(lipgloss.Left, listItems...)
	leftPane = lipgloss.NewStyle().Width(leftWidth).Render(leftPane)

	// Right Pane: Inspector
	rightWidth := m.width - leftWidth - 4
	selected := m.tables[m.cursor]
	
	detailTitle := detailTitleStyle.Render(fmt.Sprintf("DETAILS: %s", selected.Name))
	
	// Helper to render key-value pairs
	kv := func(k, v string) string {
		return lipgloss.JoinHorizontal(lipgloss.Left, labelStyle.Render(k), valueStyle.Render(v))
	}

	details := lipgloss.JoinVertical(lipgloss.Left,
		detailTitle,
		kv("Status:", selected.Status),
		kv("Region:", selected.Region),
		kv("Items:", fmt.Sprintf("%d", selected.ItemCount)),
		"",
		kv("Partition:", selected.PK),
		kv("Sort Key:", selected.SK),
		"",
		kv("Indexes:", fmt.Sprintf("%v", selected.GSIs)),
		kv("ARN:", fmt.Sprintf("arn:aws:dynamodb:%s:123456:table/%s", selected.Region, selected.Name)),
	)
	
	// Dynamic height for detail pane to fill space
	// availHeight := m.height - lipgloss.Height(header) - 5 // approx
	rightPane := detailStyle.Width(rightWidth).Height(15).Render(details)

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// --- 3. Command Bar (Bottom) ---
	cmdBar := m.input.View()
	if !m.inputMode {
		cmdBar = lipgloss.NewStyle().Foreground(subtle).Render("Press '/' to type command...")
	}
	cmdBarBox := inputStyle.Width(m.width - 2).Render(cmdBar)

	// --- Assemble ---
	// Calculate vertical gap
	contentHeight := lipgloss.Height(header) + lipgloss.Height(mainContent) + lipgloss.Height(cmdBarBox)
	gapH := m.height - contentHeight
	if gapH < 0 { gapH = 0 }
	
	return lipgloss.JoinVertical(lipgloss.Left, 
		header, 
		"\n",
		mainContent,
		strings.Repeat("\n", gapH),
		cmdBarBox,
	)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}