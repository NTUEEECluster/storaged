package main

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type listModel struct {
	list      list.Model
	executing tea.Model
}

var _ list.Item = listChoice{}

type goBackToList struct{}

func ReturnToList() tea.Msg {
	return goBackToList{}
}

type listChoice struct {
	Label   string
	Execute func() tea.Model
}

func (choice listChoice) FilterValue() string {
	return choice.Label
}

func newListModel(choices []listChoice) listModel {
	const defaultWidth = 20
	listItems := make([]list.Item, len(choices))
	for i := range choices {
		listItems[i] = choices[i]
	}
	l := list.New(listItems, listItemDelegate{}, defaultWidth, len(choices)+2)
	l.SetFilteringEnabled(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	extraKeybinding := func() []key.Binding { return []key.Binding{keybindContinue} }
	l.AdditionalShortHelpKeys = extraKeybinding
	l.AdditionalFullHelpKeys = extraKeybinding
	return listModel{
		list:      l,
		executing: nil,
	}
}

func (m listModel) Init() tea.Cmd {
	return nil
}

func (m listModel) Update(msg tea.Msg) (listModel, tea.Cmd) {
	_, ok := msg.(goBackToList)
	if ok {
		m.executing = nil
		return m, nil
	}
	if m.executing != nil {
		var cmd tea.Cmd
		m.executing, cmd = m.executing.Update(msg)
		return m, cmd
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
			i, ok := m.list.SelectedItem().(listChoice)
			if ok {
				m.executing = i.Execute()
				return m, m.executing.Init()
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m listModel) View() string {
	if m.executing != nil {
		return m.executing.View()
	}
	return m.list.View()
}

type listItemDelegate struct{}

func (d listItemDelegate) Height() int                             { return 1 }
func (d listItemDelegate) Spacing() int                            { return 0 }
func (d listItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d listItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(listChoice)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i.Label)
	if index == m.Index() {
		_, _ = fmt.Fprint(w, SelectedItemStyle.Render("> "+str))
	} else {
		_, _ = fmt.Fprint(w, ItemStyle.Render(str))
	}
}
