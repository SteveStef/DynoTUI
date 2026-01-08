package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
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
)

// --- Enums ---

type currentView int

const (
	viewLoading currentView = iota
	viewTableList
	viewTableItems
)

// --- Keys ---

type keyMap struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
	Back  key.Binding
	Help  key.Binding
	Quit  key.Binding
	Slash key.Binding
	PgDn  key.Binding
	PgUp  key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Slash, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter},
		{k.Back, k.Slash, k.Help, k.Quit},
	}
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "q"),
		key.WithHelp("q/esc", "back"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Slash: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "command"),
	),
	PgDn: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "half page down"),
	),
	PgUp: key.NewBinding(
		key.WithKeys("ctrl+u"),
		key.WithHelp("ctrl+u", "half page up"),
	),
}

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

type Item map[string]interface{}

type model struct {
	view      currentView
	width     int
	height    int
	loading   bool
	
tables    []Table
	mockItems []Item
	
tableCursor int
	itemCursor  int
	
	spinner   spinner.Model
	input     textinput.Model
	inputMode bool
	help      help.Model
	keys      keyMap
	viewport  viewport.Model
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(highlight)

	ti := textinput.New()
	ti.Placeholder = "Type a command (e.g. 'seed 50 users')..."
	ti.Prompt = "❯ "
	ti.CharLimit = 156
	ti.Width = 50

	return model{
		view:    viewLoading,
		loading: true,
		
		tables: []Table{
			{"Users", "user_id", "metadata", "us-east-1", 1250, []string{"email-index"}, "ACTIVE"},
			{"Orders", "order_id", "timestamp", "us-east-1", 45000, []string{"customer-date"}, "ACTIVE"},
			{"Products", "sku", "category", "us-west-2", 300, []string{}, "ACTIVE"},
			{"Inventory", "warehouse_id", "sku", "us-east-1", 12, []string{"sku-index"}, "UPDATING"},
		},
		
		mockItems: generateMockData(50),

		spinner: s,
		input:   ti,
		help:    help.New(),
		keys:    keys,
		viewport: viewport.New(0, 0),
	}
}

type loadedMsg struct {}

func loadData() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return loadedMsg{}
	})
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, loadData())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.input.Width = msg.Width - 10
		m.viewport.Width = m.width/2 - 4
		m.viewport.Height = m.height - 15

	case loadedMsg:
		m.loading = false
		m.view = viewTableList
		return m, nil

	case tea.KeyMsg:
		if !m.inputMode && msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if msg.String() == "/" && !m.inputMode {
			m.inputMode = true
			m.input.Focus()
			return m, textinput.Blink
		}
		
		if m.inputMode {
			switch msg.String() {
			case "enter", "esc":
				m.inputMode = false
				m.input.Blur()
				if msg.String() == "enter" {
					m.input.SetValue("") // Clear on execute
				}
			}
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "q":
			if m.view == viewTableItems {
				m.view = viewTableList
				return m, nil
			}
			return m, tea.Quit

		case "?":
			m.help.ShowAll = !m.help.ShowAll

		case "up", "k":
			if m.view == viewTableList {
				if m.tableCursor > 0 { m.tableCursor-- }
			} else if m.view == viewTableItems {
				if m.itemCursor > 0 { 
					m.itemCursor--
					m.updateViewport()
				}
			}

		case "down", "j":
			if m.view == viewTableList {
				if m.tableCursor < len(m.tables)-1 { m.tableCursor++ }
			} else if m.view == viewTableItems {
				if m.itemCursor < len(m.mockItems)-1 { 
					m.itemCursor++ 
					m.updateViewport()
				}
			}

		case "ctrl+d":
			amount := 10
			if m.view == viewTableList {
				m.tableCursor = min(m.tableCursor+amount, len(m.tables)-1)
			} else if m.view == viewTableItems {
				m.itemCursor = min(m.itemCursor+amount, len(m.mockItems)-1)
				m.updateViewport()
			}

		case "ctrl+u":
			amount := 10
			if m.view == viewTableList {
				m.tableCursor = max(m.tableCursor-amount, 0)
			} else if m.view == viewTableItems {
				m.itemCursor = max(m.itemCursor-amount, 0)
				m.updateViewport()
			}

		case "enter":
			if m.view == viewTableList {
				m.view = viewTableItems
				m.itemCursor = 0
				m.updateViewport()
			}
		}
	
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	if m.view == viewTableItems {
		var vpCmd tea.Cmd
		m.viewport, vpCmd = m.viewport.Update(msg)
		return m, vpCmd
	}

	return m, nil
}

func (m *model) updateViewport() {
	selectedItem := m.mockItems[m.itemCursor]
	b, _ := json.MarshalIndent(selectedItem, "", "  ")
	m.viewport.SetContent(highlightJSON(string(b)))
}

func (m model) View() string {
	if m.width == 0 { return "Initializing..." }

	var content string

	switch m.view {
	case viewLoading:
		content = lipgloss.Place(m.width, m.height-3, lipgloss.Center, lipgloss.Center,
			fmt.Sprintf("%s Loading tables from AWS...", m.spinner.View()),
		)
	case viewTableList:
		content = m.renderTableList()
	case viewTableItems:
		content = m.renderTableItems()
	}

	// RENDER HELP
	helpView := m.help.View(m.keys)

	// ALWAYS RENDER COMMAND BAR
	cmdBar := m.input.View()
	if !m.inputMode {
		cmdBar = lipgloss.NewStyle().Foreground(subtle).Render("Press '/' to type command...")
	}
	cmdBarBox := inputStyle.Width(m.width - 2).Render(cmdBar)

	// Calculate Gap to push bar to bottom
	contentHeight := lipgloss.Height(content) + lipgloss.Height(helpView) + lipgloss.Height(cmdBarBox) + 1
	gapH := m.height - contentHeight
	gap := ""
	if gapH > 0 {
		gap = strings.Repeat("\n", gapH)
	}

	return lipgloss.JoinVertical(lipgloss.Left, content, gap, "  "+helpView, cmdBarBox)
}

func (m model) renderTableList() string {
	header := headerStyle.Width(m.width).Render("DynamoDB TUI - Tables")

	leftWidth := int(float64(m.width) * 0.4)
	var listItems []string
	listHeader := listHeaderStyle.Width(leftWidth-2).Render("  NAME")
	listItems = append(listItems, listHeader)

	for i, t := range m.tables {
		str := fmt.Sprintf("  %s", t.Name)
		if m.tableCursor == i {
			listItems = append(listItems, listSelectedStyle.Width(leftWidth-2).Render(str))
		} else {
			listItems = append(listItems, listItemStyle.Width(leftWidth-2).Render(str))
		}
	}
	leftPane := lipgloss.JoinVertical(lipgloss.Left, listItems...)
	leftPane = lipgloss.NewStyle().Width(leftWidth).Render(leftPane)

	rightWidth := m.width - leftWidth - 4
	selected := m.tables[m.tableCursor]

	// Schema Map Visualizer
	tree := fmt.Sprintf("Table: %s\n", selected.Name)
	tree += fmt.Sprintf("├── PK: %s (S)\n", selected.PK)
	tree += fmt.Sprintf("└── SK: %s (S)\n", selected.SK)
	
	if len(selected.GSIs) > 0 {
		tree += "\nIndexes:\n"
		for i, idx := range selected.GSIs {
			isLast := i == len(selected.GSIs)-1
			prefix := "├──"
			if isLast { prefix = "└──" }
			tree += fmt.Sprintf("%s %s\n", prefix, idx)
			
			// Sub-tree for index keys
			subPrefix := "│   "
			if isLast { subPrefix = "    " }
			tree += fmt.Sprintf("%s├── PK: %s_pk\n", subPrefix, idx)
			tree += fmt.Sprintf("%s└── SK: %s_sk\n", subPrefix, idx)
		}
	} else {
		tree += "\n(No Indexes)"
	}

	details := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(accent).Render("SCHEMA MAP"),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Render(tree),
	)
	rightPane := detailStyle.Width(rightWidth).Height(15).Render(details)

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	return lipgloss.JoinVertical(lipgloss.Left, header, "\n", mainContent)
}

func (m model) renderTableItems() string {
	selectedTable := m.tables[m.tableCursor]
	header := headerStyle.Width(m.width).Render(fmt.Sprintf("Viewing: %s", selectedTable.Name))

	// Split View Dimensions
	leftWidth := int(float64(m.width) * 0.4)
	rightWidth := m.width - leftWidth - 4

	// --- LEFT PANE: Item List ---
	wPK := int(float64(leftWidth) * 0.5)
	wSK := leftWidth - wPK - 4
	
	// List Header
	tableHeader := lipgloss.JoinHorizontal(lipgloss.Left,
		itemHeaderStyle.Width(wPK).Render("PARTITION KEY"),
		itemHeaderStyle.Width(wSK).Render("SORT KEY"),
	)

	// Windowing Logic
	availableHeight := m.height - 15 
	if availableHeight < 1 { availableHeight = 1 }
	
	start := 0
	end := len(m.mockItems)
	
	if len(m.mockItems) > availableHeight {
		if m.itemCursor < availableHeight/2 {
			start = 0
			end = availableHeight
		} else if m.itemCursor >= len(m.mockItems)-availableHeight/2 {
			start = len(m.mockItems) - availableHeight
			end = len(m.mockItems)
		} else {
			start = m.itemCursor - availableHeight/2
			end = start + availableHeight
		}
	}

	var rows []string
	for i := start; i < end; i++ {
		item := m.mockItems[i]
		pk := fmt.Sprintf("%v", item["pk"])
		sk := fmt.Sprintf("%v", item["sk"])
		
		style := itemRowStyle
		if m.itemCursor == i {
			style = style.Copy().Background(highlight).Foreground(lipgloss.Color("#FFF"))
		}
		
		row := lipgloss.JoinHorizontal(lipgloss.Left,
			style.Width(wPK).Render(pk),
			style.Width(wSK).Render(sk),
		)
		rows = append(rows, row)
	}
	
	// Use JoinVertical for the list
	itemTable := lipgloss.JoinVertical(lipgloss.Left, rows...)
	leftPane := lipgloss.JoinVertical(lipgloss.Left, tableHeader, itemTable)
	leftPane = lipgloss.NewStyle().Width(leftWidth).Render(leftPane)


	// --- RIGHT PANE: JSON Inspector ---
	detailBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(subtle).
		Width(rightWidth).
		Padding(1).
		Render(lipgloss.JoinVertical(lipgloss.Left, 
			lipgloss.NewStyle().Foreground(accent).Bold(true).Render("ITEM JSON"), 
			"\n",
			m.viewport.View(),
		))

	// --- COMBINE ---
	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, detailBox)

	return lipgloss.JoinVertical(lipgloss.Left, header, "\n", mainContent)
}

func highlightJSON(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	for _, l := range lines {
		if strings.Contains(l, "\":") {
			parts := strings.SplitN(l, ":", 2)
			key := parts[0]
			rawVal := strings.TrimSpace(parts[1])
			
			// Color Key (Purple)
			key = lipgloss.NewStyle().Foreground(lipgloss.Color("#874BFD")).Render(key)
			
			// Detect Type & Color Value
			var badge string
			var valColor lipgloss.Color
			
			val := rawVal
			if strings.HasSuffix(val, ",") {
				val = strings.TrimSuffix(val, ",")
			}

			if strings.HasPrefix(val, "\"") {
				// String
				valColor = lipgloss.Color("#43BF6D") // Green
				badge = lipgloss.NewStyle().Foreground(lipgloss.Color("#222")).Background(lipgloss.Color("#43BF6D")).SetString(" S ").String()
			} else if val == "true" || val == "false" {
				// Boolean
				valColor = lipgloss.Color("#F25D94") // Pink
				badge = lipgloss.NewStyle().Foreground(lipgloss.Color("#222")).Background(lipgloss.Color("#F25D94")).SetString(" B ").String()
			} else if strings.ContainsAny(val, "0123456789") {
				// Number (simplistic check)
				valColor = lipgloss.Color("#F5C25D") // Yellow
				badge = lipgloss.NewStyle().Foreground(lipgloss.Color("#222")).Background(lipgloss.Color("#F5C25D")).SetString(" N ").String()
			} else {
				// Null or Object
				valColor = lipgloss.Color("250")
				badge = "   "
			}
			
			renderedVal := lipgloss.NewStyle().Foreground(valColor).Render(val)
			if strings.HasSuffix(rawVal, ",") {
				renderedVal += ","
			}
			
			out = append(out, fmt.Sprintf("%s: %s %s", key, badge, renderedVal))
		} else {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}
func generateMockData(n int) []Item {
	var items []Item
	for i := 0; i < n; i++ {
		items = append(items, Item{
			"pk":          fmt.Sprintf("USER#%03d", i+1),
			"sk":          "PROFILE",
			"name":        fmt.Sprintf("User %d", i+1),
			"age":         20 + (i % 50),
			"is_active":   i%3 != 0,
			"rating":      float64(i%50) / 10.0 + 1.5,
			"roles":       []string{"user", "editor"},
			"settings": map[string]interface{}{
				"theme": "dark",
				"notifications": map[string]bool{
					"email": true,
					"push":  false,
				},
			},
			"notes":       "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.",
			"last_login":  nil,
		})
	}
	return items
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
