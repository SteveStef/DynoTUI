package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
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

type Item map[string]string

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

		case "up", "k":
			if m.view == viewTableList {
				if m.tableCursor > 0 { m.tableCursor-- }
			} else if m.view == viewTableItems {
				if m.itemCursor > 0 { m.itemCursor-- }
			}

		case "down", "j":
			if m.view == viewTableList {
				if m.tableCursor < len(m.tables)-1 { m.tableCursor++ }
			} else if m.view == viewTableItems {
				if m.itemCursor < len(m.mockItems)-1 { m.itemCursor++ }
			}

		case "enter":
			if m.view == viewTableList {
				m.view = viewTableItems
				m.itemCursor = 0
			}
		}
	
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
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

	// ALWAYS RENDER COMMAND BAR
	cmdBar := m.input.View()
	if !m.inputMode {
		cmdBar = lipgloss.NewStyle().Foreground(subtle).Render("Press '/' to type command...")
	}
	cmdBarBox := inputStyle.Width(m.width - 2).Render(cmdBar)

	// Calculate Gap to push bar to bottom
	contentHeight := lipgloss.Height(content) + lipgloss.Height(cmdBarBox)
	gapH := m.height - contentHeight
	gap := ""
	if gapH > 0 {
		gap = strings.Repeat("\n", gapH)
	}

	return lipgloss.JoinVertical(lipgloss.Left, content, gap, cmdBarBox)
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
	kv := func(k, v string) string {
		return lipgloss.JoinHorizontal(lipgloss.Left, labelStyle.Render(k), valueStyle.Render(v))
	}
	details := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render("Table Details"), "",
		kv("Name:", selected.Name),
		kv("Status:", selected.Status),
		kv("Items:", fmt.Sprintf("%d", selected.ItemCount)),
		kv("PK:", selected.PK),
		kv("SK:", selected.SK),
		kv("Region:", selected.Region),
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
	availableHeight := m.height - 10 
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
		pk, sk := item["pk"], item["sk"]
		
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
	selectedItem := m.mockItems[m.itemCursor]
	
	// Pretty Print JSON
	b, _ := json.MarshalIndent(selectedItem, "", "  ")
	jsonStr := string(b)
	
	// Highlight
	jsonStr = highlightJSON(jsonStr)

	detailContent := lipgloss.NewStyle().
		Render(jsonStr)
		
	detailBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(subtle).
		Width(rightWidth).
		Padding(1).
		Render(lipgloss.JoinVertical(lipgloss.Left, 
			lipgloss.NewStyle().Foreground(accent).Bold(true).Render("ITEM JSON"), 
			"\n",
			detailContent,
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
			val := parts[1]
			
			// Color Key (Purple)
			key = lipgloss.NewStyle().Foreground(lipgloss.Color("#874BFD")).Render(key)
			// Color Value (Green)
			val = lipgloss.NewStyle().Foreground(lipgloss.Color("#43BF6D")).Render(val)
			
			out = append(out, key+":"+val)
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
			"pk": fmt.Sprintf("USER#%03d", i+1),
			"sk": "PROFILE",
			"name": fmt.Sprintf("User %d", i+1),
			"email": fmt.Sprintf("user%d@example.com", i+1),
			"address": fmt.Sprintf("%d Random St, City %d", i*5, i),
			"preferences": fmt.Sprintf(`{"theme":"dark","notifications":%t}`, i%2==0),
			"history": fmt.Sprintf(`["login","logout","purchase","view"]`),
			"metadata": fmt.Sprintf(`{"created_at":"2023-01-%02d","active":true}`, (i%30)+1),
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
