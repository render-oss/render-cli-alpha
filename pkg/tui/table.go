package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"

	renderstyle "github.com/renderinc/render-cli/pkg/style"
)

var styleSubtle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888"))

const defaultMaxWidth = 100

var defaultFilterCustomOption = CustomOption{
	Key:   "/",
	Title: "Search",
}

type CustomOption struct {
	Key      string
	Title    string
	Function func(row table.Row) tea.Cmd
}

func (o CustomOption) String() string {
	key := renderstyle.CommandKey.Render(fmt.Sprintf("[%s]", o.Key))
	return key + " " + o.Title
}

type Table[T any] struct {
	Model         table.Model
	onSelect      func(rows []table.Row) tea.Cmd
	customOptions []CustomOption

	headerMessage string
	headerStyle   lipgloss.Style

	loadData  TypedCmd[[]T]
	createRow func(T) table.Row
	data      []T
	columns   []table.Column

	tableWidth  int
	tableHeight int
}

type TableOption[T any] func(*Table[T])

func WithCustomOptions[T any](options []CustomOption) TableOption[T] {
	return func(t *Table[T]) {
		t.customOptions = options
	}
}

func WithHeader[T any](message string) TableOption[T] {
	return func(t *Table[T]) {
		t.headerMessage = message
	}
}

func NewTable[T any](
	columns []table.Column,
	loadData TypedCmd[[]T],
	createRow func(T) table.Row,
	onSelect func(rows []table.Row) tea.Cmd,
	tableOptions ...TableOption[T],
) *Table[T] {
	t := &Table[T]{
		Model: table.New(columns).
			Filtered(true).
			Focused(true).
			WithPageSize(25).
			WithBaseStyle(lipgloss.NewStyle().Align(lipgloss.Left)).
			WithTargetWidth(defaultMaxWidth).
			HighlightStyle(renderstyle.Highlight),
		tableWidth:  defaultMaxWidth,
		onSelect:    onSelect,
		loadData:    loadData,
		createRow:   createRow,
		columns:     columns,
		headerStyle: lipgloss.NewStyle().Foreground(renderstyle.ColorWarningDeprioritized),
	}

	for _, option := range tableOptions {
		option(t)
	}

	return t
}

func (t *Table[T]) Init() tea.Cmd {
	return tea.Batch(tea.Cmd(t.loadData), t.Model.Init(), tea.WindowSize())
}

func (t *Table[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case LoadDataMsg[[]T]:
		t.data = msg.Data
		rows := make([]table.Row, len(t.data))
		for i, item := range t.data {
			rows[i] = t.createRow(item)
		}
		t.Model = t.Model.WithRows(rows)
		return t, nil
	case StackSizeMsg:
		t.tableWidth = msg.Width
		t.tableHeight = msg.Height
		t.Model.WithTargetWidth(t.tableWidth)
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			return t, t.onSelect([]table.Row{t.Model.HighlightedRow()})
		default:
			if !t.Model.GetIsFilterInputFocused() {
				for _, option := range t.customOptions {
					if msg.String() == option.Key {
						return t, option.Function(t.Model.HighlightedRow())
					}
				}
			}
		}
	}

	t.Model, cmd = t.Model.Update(msg)
	return t, cmd
}

func (t *Table[T]) View() string {
	var footer string
	if len(t.customOptions) > 0 {
		var options []string
		for _, option := range t.customOptions {
			options = append(options, option.String())
		}
		options = append(options, defaultFilterCustomOption.String())
		footer = lipgloss.JoinHorizontal(
			lipgloss.Left,
			renderstyle.CommandTitle.Render("Actions:    "), // extra spaces to align with the commands shown by the stack
			strings.Join(options, " "),
		)
	}

	tableView := t.Model.View()

	if len(t.data) == 0 {
		tableView = lipgloss.Place(t.tableWidth, t.tableHeight, lipgloss.Center, lipgloss.Center, "No Results")
	}

	view := lipgloss.JoinVertical(
		lipgloss.Left,
		tableView,
	)

	if footer != "" {
		view = lipgloss.JoinVertical(
			lipgloss.Left,
			view,
			footer,
		)
	}

	if t.headerMessage != "" {
		view = lipgloss.JoinVertical(
			lipgloss.Left,
			t.headerStyle.Render(t.headerMessage),
			view,
		)
	}

	return view
}
