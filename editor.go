package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"reflect"

	tea "github.com/charmbracelet/bubbletea"
)

func openEditor(item Item, isNew bool) tea.Cmd {
	// Create temp file
	f, err := os.CreateTemp("", "dynotui-*.json")
	if err != nil {
		return func() tea.Msg { return editorFinishedMsg{err: err} }
	}
	// We verify the file closes after we write to it, so vim can open it independently
	defer f.Close()

	if item == nil {
		item = make(Item)
	}

	// Marshal item to JSON
	b, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return func() tea.Msg { return editorFinishedMsg{err: err} }
	}

	// Write to file
	if _, err := f.Write(b); err != nil {
		return func() tea.Msg { return editorFinishedMsg{err: err} }
	}

	// Open editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		// Try to find a suitable editor
		editors := []string{"nvim", "vim", "nano", "vi"}
		for _, e := range editors {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
		// Fallback if nothing found
		if editor == "" {
			editor = "nvim" 
		}
	}

	c := exec.Command(editor, f.Name())
	
	// tea.ExecProcess returns a tea.Cmd directly.
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			os.Remove(f.Name())
			return editorFinishedMsg{err: err}
		}

		// Read the file content
		content, readErr := os.ReadFile(f.Name())
		os.Remove(f.Name()) // Clean up regardless of read success

		if readErr != nil {
			return editorFinishedMsg{err: readErr}
		}

		// If file is empty, treat as cancellation
		if len(content) == 0 {
			return editorFinishedMsg{newItem: nil}
		}

		var newItem Item
		if jsonErr := json.Unmarshal(content, &newItem); jsonErr != nil {
			return editorFinishedMsg{err: jsonErr}
		}

		// If it's a new item and it's empty, treat as cancellation
		if isNew && len(newItem) == 0 {
			return editorFinishedMsg{newItem: nil}
		}
		
		if !isNew && reflect.DeepEqual(item, newItem) {
			// No changes made
			return editorFinishedMsg{newItem: nil} 
		}

		return editorFinishedMsg{newItem: newItem, isNew: isNew}
	})
}
