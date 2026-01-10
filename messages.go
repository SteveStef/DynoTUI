package main

import "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

// --- Messages ---

type tablesLoadedMsg []TableDetails // Using the struct from aws.go
type itemsLoadedMsg struct {
	items    []map[string]interface{}
	nextKey  map[string]types.AttributeValue
	isAppend bool
}
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
