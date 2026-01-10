package main

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	//"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
)

// --- Enums ---

type currentView int

const (
	viewLoading currentView = iota
	viewTableList
	viewTableItems
	viewError
	viewConfirmation
	viewDeleteConfirmation
	viewSqlConfirmation
)

// --- Model ---

type Table struct {
	Name      string
	PK        string
	SK        string
	Region    string
	ItemCount int64
	GSIs      []string
	Status    string
}

type Item map[string]interface{}

type model struct {
	view        currentView
	width       int
	height      int
	loading     bool
	tables      []Table
	items       []Item
	tableCursor int
	itemCursor  int
	modifiedItems map[int]bool
	spinner     spinner.Model
	input       textinput.Model
	inputMode   bool
	help        help.Model
	keys        keyMap
	viewport    viewport.Model
	activePane  int
	statusMessage string
	err         error
	generatedSql []string
	isScanWarning bool
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(highlight)

	ti := textinput.New()
	ti.Placeholder = "Type a command..."
	ti.Prompt = "❯ "
	ti.CharLimit = 156
	ti.Width = 50

	return model{
		view:          viewLoading,
		loading:       true,
		tables:        []Table{},
		items:         []Item{},
		modifiedItems: make(map[int]bool),
		spinner:       s,
		input:         ti,
		help:          help.New(),
		keys:          keys,
		viewport:      viewport.New(0, 0),
		statusMessage: "Loading tables from AWS...",
	}
}

type Operation struct {
	expression string
	params     []types.AttributeValue
}

