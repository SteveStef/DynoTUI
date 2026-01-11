package main

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	tea "github.com/charmbracelet/bubbletea"
)

// --- Commands ---

func loadTables() tea.Msg {
	log.Println("Starting loadTables...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("Calling ListTablesWithDetails...")
	tables, region, accountId, err := ListTablesWithDetails(ctx)
	if err != nil {
		log.Printf("ListTablesWithDetails failed: %v", err)
		return errMsg(err)
	}

	log.Printf("Successfully loaded %d tables from region %s, account %s", len(tables), region, accountId)
	return tablesLoadedMsg{tables: tables, region: region, accountId: accountId}

}

func scanTable(name string, startKey map[string]types.AttributeValue, isAppend bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		items, nextKey, err := ScanTable(ctx, name, startKey)
		if err != nil {
			return errMsg(err)
		}

		return itemsLoadedMsg{items: items, nextKey: nextKey, isAppend: isAppend}
	}
}

func saveItemCmd(tableName string, item Item) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := PutItem(ctx, tableName, item)
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

		err := DeleteItem(ctx, tableName, keyMap)
		return itemDeletedMsg{err}
	}
}

func generateSQLCmd(question string, table Table) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		result, err := InvokeBedrock(ctx, question, table)
		return sqlGeneratedMsg{result: result, err: err}
	}
}
