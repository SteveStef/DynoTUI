package main

import (
	"context"
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// --- Commands ---

func loadTables() tea.Msg {
	log.Println("Starting loadTables...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("Calling ListTablesWithDetails...")
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

func deleteItemCmd(tableName string, item Item, pkName, skName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Construct Key Map
		keyMap := make(map[string]interface{})
		if val, ok := item[pkName]; ok {
			keyMap[pkName] = val
		}
		if skName != "" {
			if val, ok := item[skName]; ok {
				keyMap[skName] = val
			}
		}

		err := DeleteItem(ctx, "us-east-1", tableName, keyMap)
		return itemDeletedMsg{err}
	}
}

func generateSQLCmd(question string, table Table) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		sqls, err := InvokeBedrock(ctx, question, table)
		return sqlGeneratedMsg{sqls: sqls, err: err}
	}
}
