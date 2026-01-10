package main

// --- Messages ---

type tablesLoadedMsg []TableDetails // Using the struct from aws.go
type itemsLoadedMsg []map[string]interface{}
type sqlGeneratedMsg struct {
	sqls []string
	err  error
}
type editorFinishedMsg struct {
	newItem Item
	err     error
	isNew   bool
}
type itemSavedMsg struct{ err error }
type itemDeletedMsg struct{ err error }
type errMsg error
