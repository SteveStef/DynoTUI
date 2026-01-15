package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
	defer f.Close()

	ctx := context.TODO()
	api, err := NewAWS(ctx)
	if err != nil {
		fmt.Printf("Failed to initialize AWS client: %v\n", err)
		os.Exit(1)
	}

	m := initialModel(api)
	p := tea.NewProgram(&m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
