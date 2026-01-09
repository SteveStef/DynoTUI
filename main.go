package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
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

// --- Enums ---

type currentView int

const (
	viewLoading currentView = iota
	viewTableList
	viewTableItems
	viewError
	viewConfirmation
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
	Edit  key.Binding
	Save  key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Slash, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter},
		{k.Back, k.Slash, k.Help, k.Quit, k.Edit, k.Save},
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
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit item"),
	),
	Save: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "save item"),
	),
}

// --- Model ---

type Table struct {
	Name      string
	PK        string
	SK        string
	Region    string
	ItemCount int64
	GSIs      []string
	Status    string
}

type Item map[string]interface{}

type model struct {
	view        currentView
	width       int
	height      int
	loading     bool
	tables      []Table
	items       []Item
	tableCursor int
	itemCursor  int
	modifiedItems map[int]bool
	spinner     spinner.Model
	input       textinput.Model
	inputMode   bool
	help        help.Model
	keys        keyMap
	viewport    viewport.Model
	activePane  int
	statusMessage string
	err         error
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(highlight)

	ti := textinput.New()
	ti.Placeholder = "Type a command..."
	ti.Prompt = "❯ "
	ti.CharLimit = 156
	ti.Width = 50

	return model{
		view:          viewLoading,
		loading:       true,
		tables:        []Table{},
		items:         []Item{},
		modifiedItems: make(map[int]bool),
		spinner:       s,
		input:         ti,
		help:          help.New(),
		keys:          keys,
		viewport:      viewport.New(0, 0),
		statusMessage: "Loading tables from AWS...",
	}
}

// --- Messages ---

type tablesLoadedMsg []TableDetails // Using the struct from aws.go
type itemsLoadedMsg []map[string]interface{}
type editorFinishedMsg struct {
	newItem Item
	err     error
}
type itemSavedMsg struct{ err error }
type errMsg error

// --- Commands ---





func loadTables() tea.Msg {

	log.Println("Starting loadTables...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()



	log.Println("Calling ListTablesWithDetails...")

	// Region is hardcoded for now, mimicking original aws.go logic.

	tables, err := ListTablesWithDetails(ctx, "us-east-1")

	if err != nil {

		log.Printf("ListTablesWithDetails failed: %v", err)

		return errMsg(err)

	}

	log.Printf("Successfully loaded %d tables", len(tables))

	return tablesLoadedMsg(tables)

}



func scanTable(name string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		items, err := ScanTable(ctx, "us-east-1", name)
		if err != nil {
			return errMsg(err)
		}
		return itemsLoadedMsg(items)
	}
}

func saveItemCmd(tableName string, item Item) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := PutItem(ctx, "us-east-1", tableName, item)
		return itemSavedMsg{err}
	}
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, func() tea.Msg { return loadTables() })
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.input.Width = msg.Width - 10
		m.viewport.Width = m.width/2 - 4
		m.viewport.Height = m.height - 15

	case tablesLoadedMsg:
		m.loading = false
		m.view = viewTableList
		m.tables = make([]Table, len(msg))
		for i, t := range msg {
			m.tables[i] = Table{
				Name:      t.Name,
				PK:        t.PK,
				SK:        t.SK,
				Region:    t.Region,
				ItemCount: t.ItemCount,
				GSIs:      t.GSIs,
				Status:    t.Status,
			}
		}
		return m, nil

	case itemsLoadedMsg:
		m.loading = false
		m.view = viewTableItems
		newItems := make([]Item, len(msg))
		for i, item := range msg {
			newItems[i] = Item(item)
		}
		m.items = newItems
		m.modifiedItems = make(map[int]bool)
		m.itemCursor = 0
		m.activePane = 0
		m.updateViewport()
		return m, nil

	case editorFinishedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.view = viewError
		} else if msg.newItem != nil {
			m.items[m.itemCursor] = msg.newItem
			m.modifiedItems[m.itemCursor] = true
			m.updateViewport()
		}
		return m, nil

	case itemSavedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.view = viewError
		} else {
			// Success! Clear modified flag.
			delete(m.modifiedItems, m.itemCursor)
			m.view = viewTableItems
			m.updateViewport()
		}
		return m, nil

	case errMsg:
		m.err = msg
		m.loading = false
		m.view = viewError
		return m, nil

	case tea.KeyMsg:
		if m.view == viewError {
			// Allow any key to go back
			m.view = viewTableList
			m.err = nil
			return m, nil
		}

		if m.view == viewConfirmation {
			switch msg.String() {
			case "y", "Y", "enter":
				m.loading = true
				m.view = viewLoading
				m.statusMessage = "Saving item to DynamoDB..."
				return m, saveItemCmd(m.tables[m.tableCursor].Name, m.items[m.itemCursor])
			case "n", "N", "esc":
				m.view = viewTableItems
				return m, nil
			default:
				return m, nil
			}
		}

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
				m.items = []Item{} // Clear items to save memory
				m.activePane = 0
				return m, nil
			}
			return m, tea.Quit

		case "l", "right":
			if m.view == viewTableItems {
				m.activePane = 1
			}
		case "h", "left":
			if m.view == viewTableItems {
				m.activePane = 0
			}

		case "?":
			m.help.ShowAll = !m.help.ShowAll

		case "up", "k":
			if m.view == viewTableList {
				if m.tableCursor > 0 {
					m.tableCursor--
				}
			} else if m.view == viewTableItems {
				if m.activePane == 0 {
					if m.itemCursor > 0 {
						m.itemCursor--
						m.updateViewport()
					}
				} else {
					m.viewport.LineUp(1)
				}
			}

		case "down", "j":
			if m.view == viewTableList {
				if m.tableCursor < len(m.tables)-1 {
					m.tableCursor++
				}
			} else if m.view == viewTableItems {
				if m.activePane == 0 {
					if m.itemCursor < len(m.items)-1 {
						m.itemCursor++
						m.updateViewport()
					}
				} else {
					m.viewport.LineDown(1)
				}
			}

		case "ctrl+d":
			amount := 10
			if m.view == viewTableList {
				m.tableCursor = min(m.tableCursor+amount, len(m.tables)-1)
			} else if m.view == viewTableItems {
				if m.activePane == 0 {
					m.itemCursor = min(m.itemCursor+amount, len(m.items)-1)
					m.updateViewport()
				} else {
					m.viewport.HalfViewDown()
				}
			}

		case "ctrl+u":
			amount := 10
			if m.view == viewTableList {
				m.tableCursor = max(m.tableCursor-amount, 0)
			} else if m.view == viewTableItems {
				if m.activePane == 0 {
					m.itemCursor = max(m.itemCursor-amount, 0)
					m.updateViewport()
				} else {
					m.viewport.HalfViewUp()
				}
			}

		case "enter", " ":
			if m.view == viewTableList && msg.String() == "enter" {
				if len(m.tables) > 0 {
					m.loading = true
					m.view = viewLoading
					m.statusMessage = fmt.Sprintf("Scanning %s...", m.tables[m.tableCursor].Name)
					return m, scanTable(m.tables[m.tableCursor].Name)
				}
			} else if m.view == viewTableItems {
				m.activePane = 1
			}

		case "e", "E":
			log.Printf("Edit key pressed. View: %v, Items: %d", m.view, len(m.items))
			if m.view == viewTableItems && len(m.items) > 0 {
				log.Println("Opening editor...")
				return m, openEditor(m.items[m.itemCursor])
			}

		case "s", "S":
			if m.view == viewTableItems && len(m.items) > 0 {
				m.view = viewConfirmation
				return m, nil
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
	if len(m.items) == 0 {
		m.viewport.SetContent("No items found.")
		return
	}
	selectedItem := m.items[m.itemCursor]
	b, _ := json.MarshalIndent(selectedItem, "", "  ")
	m.viewport.SetContent(highlightJSON(string(b)))
}

func (m model) View() string {
	if m.width == 0 { return "Initializing..." }

	var content string

	switch m.view {
	case viewLoading:
		content = lipgloss.Place(m.width, m.height-3, lipgloss.Center, lipgloss.Center,
			fmt.Sprintf("%s %s", m.spinner.View(), m.statusMessage),
		)
	case viewTableList:
		content = m.renderTableList()
	case viewTableItems:
		content = m.renderTableItems()
	case viewConfirmation:
		question := lipgloss.NewStyle().Bold(true).Render("Are you sure you want to save this item to DynamoDB?")
		warning := lipgloss.NewStyle().Foreground(warning).Render("This will overwrite the existing item.")
		controls := lipgloss.NewStyle().Foreground(subtle).Render("(y/enter to confirm, n/esc to cancel)")
		
		content = lipgloss.Place(m.width, m.height-3, lipgloss.Center, lipgloss.Center,
			dialogBoxStyle.Render(
				lipgloss.JoinVertical(lipgloss.Center,
					question,
					"",
					warning,
					"",
					controls,
				),
			),
		)
	case viewError:
		content = lipgloss.Place(m.width, m.height-3, lipgloss.Center, lipgloss.Center,
			lipgloss.JoinVertical(lipgloss.Center,
				lipgloss.NewStyle().Foreground(warning).Bold(true).Render("ERROR"),
				"",
				lipgloss.NewStyle().Width(m.width/2).Align(lipgloss.Center).Render(fmt.Sprintf("%v", m.err)),
				"",
				lipgloss.NewStyle().Foreground(subtle).Render("Press any key to continue"),
			),
		)
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
	
	// Details Pane
	if len(m.tables) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left, header, "\n", "  No tables found.")
	}
	
	selected := m.tables[m.tableCursor]

	// Schema Map Visualizer
	tree := fmt.Sprintf("Table: %s (%s)\n", selected.Name, selected.Status)
	tree += fmt.Sprintf("Items: %d\n", selected.ItemCount)
	tree += fmt.Sprintf("├── PK: %s (HASH)\n", selected.PK)
	if selected.SK != "" {
		tree += fmt.Sprintf("└── SK: %s (RANGE)\n", selected.SK)
	} else {
		tree += "└── (No Sort Key)\n"
	}
	
	if len(selected.GSIs) > 0 {
		tree += "\nIndexes (GSI):\n"
		for i, idx := range selected.GSIs {
			isLast := i == len(selected.GSIs)-1
			prefix := "├──"
			if isLast { prefix = "└──" }
			tree += fmt.Sprintf("%s %s\n", prefix, idx)
		}
	} else {
		tree += "\n(No Global Indexes)"
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
	if len(m.tables) == 0 {
		return "No tables available."
	}
	selectedTable := m.tables[m.tableCursor]
	header := headerStyle.Width(m.width).Render(fmt.Sprintf("Viewing: %s", selectedTable.Name))

	// Split View Dimensions
	leftWidth := int(float64(m.width) * 0.4)
	rightWidth := m.width - leftWidth - 4

	// --- LEFT PANE: Item List ---
	
	// Just use generic ID column since we don't know PK/SK
	wPK := leftWidth - 4
	
	// List Header
	tableHeader := itemHeaderStyle.Width(wPK).Render("ITEMS (Summary)")

	// Windowing Logic
	availableHeight := m.height - 15 
	if availableHeight < 1 { availableHeight = 1 }
	
	start := 0
	end := len(m.items)
	
	if len(m.items) > availableHeight {
		if m.itemCursor < availableHeight/2 {
			start = 0
			end = availableHeight
		} else if m.itemCursor >= len(m.items)-availableHeight/2 {
			start = len(m.items) - availableHeight
			end = len(m.items)
		} else {
			start = m.itemCursor - availableHeight/2
			end = start + availableHeight
		}
	}

	var rows []string
	if len(m.items) == 0 {
		rows = append(rows, itemRowStyle.Render("No items found or empty table."))
	}

	for i := start; i < end; i++ {
		item := m.items[i]
		
		// Grab first two keys to display as a summary
		var keys []string
		for k := range item {
			keys = append(keys, k)
		}
		sort.Strings(keys) // Consistent order
		
		summary := ""
		
		// Add modification indicator
		if m.modifiedItems[i] {
			summary += "[+] "
		}

		if len(keys) > 0 { summary += fmt.Sprintf("%s=%v ", keys[0], item[keys[0]]) }
		if len(keys) > 1 { summary += fmt.Sprintf("%s=%v", keys[1], item[keys[1]]) }
		if summary == "" { summary = "{empty}" }

		// Truncate
		if len(summary) > wPK-2 {
			summary = summary[:wPK-2] + ".."
		}

		style := itemRowStyle
		if m.itemCursor == i {
			style = style.Copy().Background(highlight).Foreground(lipgloss.Color("#FFF"))
		}
		
		rows = append(rows, style.Width(wPK).Render(summary))
	}
	
	// Use JoinVertical for the list
	itemTable := lipgloss.JoinVertical(lipgloss.Left, rows...)
	leftPane := lipgloss.JoinVertical(lipgloss.Left, tableHeader, itemTable)
	leftPane = lipgloss.NewStyle().Width(leftWidth).Render(leftPane)


	// --- RIGHT PANE: JSON Inspector ---
	detailBorderColor := subtle
	if m.activePane == 1 {
		detailBorderColor = highlight
	}
	
	detailTitle := "ITEM JSON"
	if m.modifiedItems[m.itemCursor] {
		detailTitle = "ITEM JSON (MODIFIED - Not Synced)"
	}

	detailBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(detailBorderColor).
		Width(rightWidth).
		Padding(1).
		Render(lipgloss.JoinVertical(lipgloss.Left, 
			lipgloss.NewStyle().Foreground(accent).Bold(true).Render(detailTitle), 
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
			var valColor lipgloss.Color
			
			val := rawVal
			if strings.HasSuffix(val, ",") {
				val = strings.TrimSuffix(val, ",")
			}

			if strings.HasPrefix(val, "\"") {
				// String
				valColor = lipgloss.Color("#43BF6D") // Green
			} else if val == "true" || val == "false" {
				// Boolean
				valColor = lipgloss.Color("#F25D94") // Pink
			} else if strings.ContainsAny(val, "0123456789") {
				// Number
				valColor = lipgloss.Color("#F5C25D") // Yellow
			} else {
				// Null or Object
				valColor = lipgloss.Color("250")
			}
			
			renderedVal := lipgloss.NewStyle().Foreground(valColor).Render(val)
			if strings.HasSuffix(rawVal, ",") {
				renderedVal += ","
			}
			
			out = append(out, fmt.Sprintf("%s: %s", key, renderedVal))
		} else {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}
func main() {
	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
	defer f.Close()

	m := initialModel()
	p := tea.NewProgram(&m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
