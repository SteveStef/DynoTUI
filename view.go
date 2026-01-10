package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
	case viewDeleteConfirmation:
		question := lipgloss.NewStyle().Bold(true).Render("Are you sure you want to DELETE this item?")
		warning := lipgloss.NewStyle().Foreground(warning).Render("This action cannot be undone.")
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
	case viewSqlConfirmation:
		title := lipgloss.NewStyle().Bold(true).Foreground(highlight).Render("Execute Generated SQL?")
		
		// Join multiple statements for display
		joinedSql := strings.Join(m.generatedSql, "\n\n")
		
		sqlText := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Width(m.width - 20).Render(joinedSql)
		controls := lipgloss.NewStyle().Foreground(subtle).Render("(y/enter to execute, n/esc to cancel)")
		
		var contentComponents []string
		contentComponents = append(contentComponents, title, "", sqlText, "")
		
		if m.isScanWarning {
			scanWarn := lipgloss.NewStyle().Foreground(warning).Bold(true).Render("⚠ WARNING: This query may result in a FULL TABLE SCAN!")
			contentComponents = append(contentComponents, scanWarn, "")
		}
		
		contentComponents = append(contentComponents, controls)

		content = lipgloss.Place(m.width, m.height-3, lipgloss.Center, lipgloss.Center,
			dialogBoxStyle.Render(
				lipgloss.JoinVertical(lipgloss.Center, contentComponents...),
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

func (m model) renderHeader(title string) string {
    regionText := fmt.Sprintf("Region: %s", m.Region)
    if m.Region == "" { regionText = "Region: Loading..." }

	accountText := fmt.Sprintf("Account: %s", m.AccountId)
	if m.AccountId == "" { accountText = "Account: Loading..." }

	infoText := fmt.Sprintf("%s | %s", accountText, regionText)

    // Left part
    left := lipgloss.NewStyle().
        Background(highlight).
        Foreground(lipgloss.Color("#FFFDF5")).
        Bold(true).
        Padding(0, 1).
        Render(title)

    // Right part
    right := lipgloss.NewStyle().
        Background(highlight).
        Foreground(lipgloss.Color("#FFFDF5")).
        Padding(0, 1).
        Render(infoText)
        
    // Spacer
    w := m.width - lipgloss.Width(left) - lipgloss.Width(right)
    if w < 0 { w = 0 }
    spacer := lipgloss.NewStyle().Background(highlight).Width(w).Render("")
    
    return lipgloss.JoinHorizontal(lipgloss.Top, left, spacer, right)
}

func (m model) renderTableList() string {
	header := m.renderHeader("DynamoDB TUI - Tables")

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
	header := m.renderHeader(fmt.Sprintf("Viewing: %s", selectedTable.Name))

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
		if strings.Contains(l, ":") {
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