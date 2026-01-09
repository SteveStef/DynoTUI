package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

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
			if msg.isNew {
				m.items = append(m.items, msg.newItem)
				m.itemCursor = len(m.items) - 1
				m.modifiedItems[m.itemCursor] = true
				m.updateViewport()
			} else {
				m.items[m.itemCursor] = msg.newItem
				m.modifiedItems[m.itemCursor] = true
				m.updateViewport()
			}
		}
		return m, nil

	case itemSavedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.view = viewError
		} else {
			// Success! Clear modified flag for the saved item
			delete(m.modifiedItems, m.itemCursor)

			// Deduplicate: Remove OTHER items with the same PK/SK
			savedItem := m.items[m.itemCursor]
			currentTable := m.tables[m.tableCursor]

			// Helper to get key value
			getKeyVal := func(itm Item, key string) interface{} {
				if val, ok := itm[key]; ok {
					return val
				}
				return nil
			}

			pkVal := getKeyVal(savedItem, currentTable.PK)
			skVal := getKeyVal(savedItem, currentTable.SK)

			var newItems []Item
			// We need to rebuild modifiedItems because indices will shift
			newModifiedItems := make(map[int]bool)
			
			// We track the new index of the current cursor
			newCursor := m.itemCursor
			
			// Source index vs Destination index
			dstIdx := 0
			
			for srcIdx, item := range m.items {
				if srcIdx == m.itemCursor {
					// Always keep the item we just saved
					newItems = append(newItems, item)
					// If it was modified (it shouldn't be now), we'd map it.
					// newCursor tracks where this lands
					newCursor = dstIdx
					dstIdx++
					continue
				}

				// Check for key match
				itemPK := getKeyVal(item, currentTable.PK)
				match := (itemPK == pkVal)
				
				if match && currentTable.SK != "" {
					itemSK := getKeyVal(item, currentTable.SK)
					match = (itemSK == skVal)
				}

				if match {
					// Duplicate found! Skip it.
					// Do NOT increment dstIdx.
				} else {
					// Keep this item
					newItems = append(newItems, item)
					// Preserve modified status
					if m.modifiedItems[srcIdx] {
						newModifiedItems[dstIdx] = true
					}
					dstIdx++
				}
			}

			m.items = newItems
			m.modifiedItems = newModifiedItems
			m.itemCursor = newCursor

			m.view = viewTableItems
			m.updateViewport()
		}
		return m, nil

	case itemDeletedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.view = viewError
		} else {
			// Remove the item from the list
			if len(m.items) > 0 {
				m.items = append(m.items[:m.itemCursor], m.items[m.itemCursor+1:]...)
				// Adjust cursor if necessary
				if m.itemCursor >= len(m.items) && m.itemCursor > 0 {
					m.itemCursor--
				}
				// Also clear/rebuild modifiedItems map (simplification: clear it or we have to shift)
				// For simplicity, we'll clear the modified flag for the deleted index, 
				// but strictly we should shift keys > cursor down by 1.
				// Let's rebuild it properly.
				newModified := make(map[int]bool)
				for k, v := range m.modifiedItems {
					if k < m.itemCursor {
						newModified[k] = v
					} else if k > m.itemCursor {
						newModified[k-1] = v
					}
				}
				m.modifiedItems = newModified
			}
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

		if m.view == viewDeleteConfirmation {
			switch msg.String() {
			case "y", "Y", "enter":
				m.loading = true
				m.view = viewLoading
				m.statusMessage = "Deleting item from DynamoDB..."
				t := m.tables[m.tableCursor]
				return m, deleteItemCmd(t.Name, m.items[m.itemCursor], t.PK, t.SK)
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
				return m, openEditor(m.items[m.itemCursor], false)
			}

		case "a", "A":
			if m.view == viewTableItems {
				return m, openEditor(nil, true)
			}

		case "s", "S":
			if m.view == viewTableItems && len(m.items) > 0 {
				m.view = viewConfirmation
				return m, nil
			}

		case "d", "D":
			if m.view == viewTableItems && len(m.items) > 0 {
				m.view = viewDeleteConfirmation
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
