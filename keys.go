package main

import (
	"github.com/charmbracelet/bubbles/key"
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
	Add   key.Binding
	Delete key.Binding
	LoadMore key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Slash, k.Edit, k.Add, k.Delete, k.LoadMore, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.LoadMore},
		{k.Back, k.Slash, k.Help, k.Quit, k.Edit, k.Save, k.Add, k.Delete},
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
		key.WithHelp("e", "edits the item"),
	),
	Save: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "save item"),
	),
	Add: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "makes a new item"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete an item"),
	),
	LoadMore: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "load next page"),
	),
}
