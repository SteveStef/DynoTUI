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
		content = lipgloss.Place(m.width, m.height-4, lipgloss.Center, lipgloss.Center,
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
		
		content = lipgloss.Place(m.width, m.height-4, lipgloss.Center, lipgloss.Center,
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
		
		content = lipgloss.Place(m.width, m.height-4, lipgloss.Center, lipgloss.Center,
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
		
		// Viewport Frame
		vpWidth := m.sqlViewport.Width
		vpStyle := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(subtle).
			Padding(0, 1).
			Width(vpWidth)
		
		sqlText := vpStyle.Render(m.sqlViewport.View())
		
		controls := lipgloss.NewStyle().Foreground(subtle).Render("(y/enter to execute, n/esc to cancel, j/k to scroll)")
		
		var contentComponents []string
		contentComponents = append(contentComponents, title, sqlText)
		
		if m.isScanWarning {
			scanWarn := lipgloss.NewStyle().Foreground(warning).Bold(true).Render("⚠ WARNING: This query may result in a FULL TABLE SCAN!")
			contentComponents = append(contentComponents, scanWarn, "")
		}
		
		contentComponents = append(contentComponents, controls)

		// Calculate dialog width based on viewport width (plus padding/borders)
		dialogWidth := vpWidth + 6

		content = lipgloss.Place(m.width, m.height-4, lipgloss.Center, lipgloss.Center,
			dialogBoxStyle.Width(dialogWidth).Align(lipgloss.Center).Render(
				lipgloss.JoinVertical(lipgloss.Center, contentComponents...),
			),
		)

	case viewBulkConfirmation:
		title := lipgloss.NewStyle().Bold(true).Foreground(highlight).Render("Confirm Bulk Action")
		
		count := len(m.pendingPlanItems)
		info := fmt.Sprintf("Step 1 Complete: Found %d items.", count)
		
		var actionStr string
		if m.llmResult.Plan != nil && m.llmResult.Plan.Write != nil {
			actionStr = fmt.Sprintf("Step 2: %s ALL %d items?", strings.ToUpper(m.llmResult.Plan.Write.Action), count)
		} else {
			actionStr = "Proceed with updates?"
		}

		infoText := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(info)
		actionText := lipgloss.NewStyle().Foreground(warning).Bold(true).Render(actionStr)
		controls := lipgloss.NewStyle().Foreground(subtle).Render("(y/enter to execute, n/esc to cancel)")
		
		content = lipgloss.Place(m.width, m.height-4, lipgloss.Center, lipgloss.Center,
			dialogBoxStyle.Render(
				lipgloss.JoinVertical(lipgloss.Center,
					title,
					"",
					infoText,
					"",
					actionText,
					"",
					controls,
				),
			),
		)

	case viewError:
		content = lipgloss.Place(m.width, m.height-4, lipgloss.Center, lipgloss.Center,
			lipgloss.JoinVertical(lipgloss.Center,
				lipgloss.NewStyle().Foreground(warning).Bold(true).Render("ERROR"),
				"",
				lipgloss.NewStyle().Width(m.width/2).Align(lipgloss.Center).Render(fmt.Sprintf("%v", m.err)),
				"",
				lipgloss.NewStyle().Foreground(subtle).Render("Press any key to continue"),
			),
		)
	}

	// FOOTER LAYOUT
	// 1. Command Bar (Input) or Status Hint
	var bottomBar string
	if m.inputMode {
		bottomBar = inputStyle.Width(m.width - 2).Render(m.input.View())
	} else {
		// Render a nice status bar
		// Mode | Context | Help Hint
		modeStr := " EXPLORE "
		if m.view == viewTableItems {
			modeStr = " BROWSE "
		}
		
		mode := statusKeyStyle.Render(modeStr)
		
		accountID := m.AccountId
		if accountID == "" { accountID = "Loading..." }
		
		contextStr := fmt.Sprintf("Account: %s | Region: %s", accountID, m.Region)
		if len(m.tables) > 0 && m.tableCursor < len(m.tables) {
			t := m.tables[m.tableCursor]
			contextStr += fmt.Sprintf(" | Table: %s", t.Name)
		}
		
		context := statusValStyle.Width(m.width - lipgloss.Width(mode)).Render(contextStr)
		bottomBar = lipgloss.JoinHorizontal(lipgloss.Top, mode, context)
	}

	// Calculate Gap to push bar to bottom
	contentHeight := lipgloss.Height(content) + lipgloss.Height(bottomBar)
	gapH := m.height - contentHeight
	if gapH < 0 { gapH = 0 }
	gap := strings.Repeat("\n", gapH)

	return lipgloss.JoinVertical(lipgloss.Left, content, gap, bottomBar)
}

func (m model) renderHeader(title string) string {
    regionText := fmt.Sprintf("Region: %s", m.Region)
    if m.Region == "" { regionText = "Region: Loading..." }

	accountText := fmt.Sprintf("Account: %s", m.AccountId)
	if m.AccountId == "" { accountText = "Account: Loading..." }

	infoText := fmt.Sprintf("%s | %s", accountText, regionText)

    // Left part
    left := lipgloss.NewStyle().
        Background(primary).
        Foreground(textLight).
        Bold(true).
        Padding(0, 1).
        Render(title)

    // Right part
    right := lipgloss.NewStyle().
        Background(primary).
        Foreground(textLight).
        Padding(0, 1).
        Render(infoText)
        
    // Spacer
    w := m.width - lipgloss.Width(left) - lipgloss.Width(right)
    if w < 0 { w = 0 }
    spacer := lipgloss.NewStyle().Background(primary).Width(w).Render("")
    
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
			// Selected Item
			listItems = append(listItems, listSelectedStyle.Width(leftWidth).Render(str))
		} else {
			// Normal Item
			listItems = append(listItems, listItemStyle.Width(leftWidth).Render(str))
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
	tree += fmt.Sprintf("├── PK: %s (%s, HASH)\n", selected.PK, selected.PKType)
	if selected.SK != "" {
		tree += fmt.Sprintf("└── SK: %s (%s, RANGE)\n", selected.SK, selected.SKType)
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
		lipgloss.NewStyle().Bold(true).Foreground(secondary).Render("SCHEMA MAP"),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Render(tree),
	)
	schemaBox := detailStyle.Width(rightWidth).Render(details)

	// Help Box
	helpBox := m.renderHelpBox(rightWidth)

	rightPane := lipgloss.JoinVertical(lipgloss.Left, schemaBox, "\n", helpBox)

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	return lipgloss.JoinVertical(lipgloss.Left, header, "\n", mainContent)
}

func (m model) renderHelpBox(width int) string {
	keyStyle := lipgloss.NewStyle().Foreground(primary).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(textDim)
	
	row := func(k, d string) string {
		// Padding logic
		kStr := keyStyle.Render(fmt.Sprintf("%-7s", k))
		dStr := descStyle.Render(d)
		return fmt.Sprintf("%s %s", kStr, dStr)
	}

	col1Width := width / 2
	
	makeRow := func(k1, d1, k2, d2 string) string {
		left := row(k1, d1)
		right := row(k2, d2)
		// Calculate spacing
		gap := col1Width - lipgloss.Width(left)
		if gap < 2 { gap = 2 }
		return lipgloss.JoinHorizontal(lipgloss.Left, left, strings.Repeat(" ", gap), right)
	}

	title := lipgloss.NewStyle().Foreground(secondary).Bold(true).Render("HELP & SHORTCUTS")
	desc := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render("AI-powered DynamoDB Explorer")
	
	aboutText := "DynoTUI allows you to explore and manage your DynamoDB tables with ease. It leverages AI to let you query your data using natural language, making complex PartiQL queries accessible to everyone."
	about := lipgloss.NewStyle().Foreground(textDim).Width(width - 4).Render(aboutText)

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		desc,
		"",
		about,
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Render("[ GLOBAL ]"),
		makeRow("/", "AI Query", "r", "Refresh"),
		makeRow("t", "Theme", "q", "Back/Quit"),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Render("[ NAVIGATION ]"),
		makeRow("k/↑", "Up", "j/↓", "Down"),
		makeRow("ctrl+u", "Page Up", "ctrl+d", "Page Down"),
		makeRow("Enter", "Select", "p", "Load More"),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Render("[ EDITING ]"),
		makeRow("a", "Add New", "e", "Edit Item"),
		makeRow("d", "Delete", "s", "Save Item"),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Render("[ QUERY EXAMPLES ]"),
		lipgloss.NewStyle().Foreground(textDim).Render(`• "Find items where status is 'active'"`),
		lipgloss.NewStyle().Foreground(textDim).Render(`• "Update all items where age > 50, set status to 'retired'"`),
		lipgloss.NewStyle().Foreground(textDim).Render(`• "Add a field 'category' with value 'new' to all items"`),
		lipgloss.NewStyle().Foreground(textDim).Render(`• "Delete all items where expired is true"`),
	)

	return detailStyle.Width(width).Render(content)
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
	
	// Column Calculations
	// We want 3 columns: PK, SK, Info
	// If SK is empty, maybe just PK and Info
	
	colPadding := 1
	availWidth := leftWidth - 5 // borders/padding
	
	var col1W, col2W, col3W int
	
	hasSK := selectedTable.SK != ""
	
	if hasSK {
		col1W = int(float64(availWidth) * 0.3)
		col2W = int(float64(availWidth) * 0.3)
		col3W = availWidth - col1W - col2W - (colPadding * 2)
	} else {
		col1W = int(float64(availWidth) * 0.4)
		col2W = 0
		col3W = availWidth - col1W - (colPadding * 1)
	}
	
	// Header Row
	pkHeader := selectedTable.PK
	if len(pkHeader) > col1W { pkHeader = pkHeader[:col1W] }
	
	skHeader := selectedTable.SK
	if len(skHeader) > col2W { skHeader = skHeader[:col2W] }
	
	otherHeader := "Info"
	
	headerStyle := lipgloss.NewStyle().Foreground(textDim).Bold(true)
	
	h1 := headerStyle.Width(col1W).Render(pkHeader)
	h2 := ""
	if hasSK {
		h2 = headerStyle.Width(col2W).PaddingLeft(colPadding).Render(skHeader)
	}
	h3 := headerStyle.Width(col3W).PaddingLeft(colPadding).Render(otherHeader)
	
	colHeader := lipgloss.JoinHorizontal(lipgloss.Left, h1, h2, h3)
	listHeader := itemHeaderStyle.Width(leftWidth-2).Render(colHeader)

	// Windowing Logic
	availableHeight := m.height - 11 
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
		rows = append(rows, itemRowStyle.Render("No items found."))
	}
	
	for i := start; i < end; i++ {
		item := m.items[i]
		isSelected := m.itemCursor == i
		
		// Extract Values
		pkVal := fmt.Sprintf("%v", item[selectedTable.PK])
		skVal := ""
		if hasSK {
			skVal = fmt.Sprintf("%v", item[selectedTable.SK])
		}
		
		// Find "Other" (first key that isn't PK or SK) - Deterministic
		otherVal := ""
		var keys []string
		for k := range item {
			keys = append(keys, k)
		}
		sort.Strings(keys) // Ensure stable order

		for _, k := range keys {
			if k != selectedTable.PK && k != selectedTable.SK {
				val := fmt.Sprintf("%v", item[k])
				val = strings.ReplaceAll(val, "\n", " ") // Ensure no newlines
				otherVal = fmt.Sprintf("%s: %s", k, val)
				break
			}
		}
		
		// Truncate
		if len(pkVal) > col1W && col1W > 1 { pkVal = pkVal[:col1W-1] + "…" }
		if len(skVal) > col2W && col2W > 1 { skVal = skVal[:col2W-1] + "…" }
		
		limit := col3W - 2
		if limit < 0 { limit = 0 }
		if len(otherVal) > col3W { 
			if col3W > 5 {
				otherVal = otherVal[:col3W-5] + "…" 
			} else if col3W > 1 {
				otherVal = otherVal[:col3W-1] + "…"
			} else {
				otherVal = ""
			}
		}
		
		// Base row style
		style := tableRowStyle
		if isSelected {
			style = tableSelectedRowStyle
		}

		// Row content with colors
		pkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
		skStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
		infoStyle := lipgloss.NewStyle().Foreground(textDim)

		if isSelected {
			skStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
			infoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
		}

		// Cursor
		cursor := "  "
		if isSelected {
			cursor = "▸ "
		}

		c1 := pkStyle.Width(col1W).Render(pkVal)
		c2 := ""
		if hasSK {
			c2 = skStyle.Width(col2W).PaddingLeft(colPadding).Render(skVal)
		}
		c3 := infoStyle.Width(col3W).PaddingLeft(colPadding).Render(otherVal)
		
		rowContent := lipgloss.JoinHorizontal(lipgloss.Left, cursor, c1, c2, c3)
		rows = append(rows, style.Width(leftWidth).Render(rowContent))
	}	
	
	// Use JoinVertical for the list
	itemTable := lipgloss.JoinVertical(lipgloss.Left, rows...)
	leftPane := lipgloss.JoinVertical(lipgloss.Left, listHeader, itemTable)
	leftPane = lipgloss.NewStyle().Width(leftWidth).Render(leftPane)


	// --- RIGHT PANE: JSON Inspector ---
	var detailBorderColor lipgloss.TerminalColor = subtle
	if m.activePane == 1 {
		detailBorderColor = primary // Use primary color for focus
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
			lipgloss.NewStyle().Foreground(secondary).Bold(true).Render(detailTitle), 
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
			
			// Color Key (Purple/Primary)
			key = lipgloss.NewStyle().Foreground(primary).Render(key)
			
			// Detect Type & Color Value
			var valColor lipgloss.Color
			
			val := rawVal
			if strings.HasSuffix(val, ",") {
				val = strings.TrimSuffix(val, ",")
			}

			if strings.HasPrefix(val, "\"") {
				// String - Green (Secondary)
				valColor = secondary
			} else if val == "true" || val == "false" {
				// Boolean - Red/Pink (Alert)
				valColor = alert
			} else if strings.ContainsAny(val, "0123456789") {
				// Number - Yellow (Custom)
				valColor = lipgloss.Color("#F5C25D")
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