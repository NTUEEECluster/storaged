package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	ErrorTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF0000"))
	ErrorStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF7777"))
	LoadingStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00"))
	OKStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))

	ItemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	SelectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	InputHeaderStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
)

var (
	keybindPrev = key.NewBinding(
		key.WithHelp("ctrl+p/shift+tab", "previous input"),
		key.WithKeys("ctrl+p", "shift+tab"),
	)
	keybindNext = key.NewBinding(
		key.WithHelp("ctrl+n/tab", "next input"),
		key.WithKeys("ctrl+n", "tab"),
	)
	keybindSubmit = key.NewBinding(
		key.WithHelp("enter", "submit request"),
		key.WithKeys("enter"),
	)
	keybindContinue = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "continue"),
	)
	keybindCancel = key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	)
)

type storageModel struct {
	quotaDisplay webRequestModel
	nextChoices  listModel
}

func newStorageModel(hostURL string, username string) storageModel {
	return storageModel{
		quotaDisplay: newQuotaRequest(hostURL, username),
		nextChoices: newListModel([]listChoice{
			{"Create New Folder", func() tea.Model { return NewQuotaModel(hostURL, "Create", false) }},
			{"Update Quota for Folder", func() tea.Model { return NewQuotaModel(hostURL, "Update", false) }},
			{"Delete Folder", func() tea.Model { return NewQuotaModel(hostURL, "Delete", true) }},
			{"Quit", func() tea.Model { return quitModel{} }},
		}),
	}
}

func (m storageModel) Init() tea.Cmd {
	return m.quotaDisplay.Init()
}

func (m storageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	currentReq, reqCmd := m.quotaDisplay.Update(msg)
	m.quotaDisplay = currentReq.(webRequestModel)
	var redoReqCmd tea.Cmd
	switch msg := msg.(type) {
	case goBackToList:
		m.quotaDisplay, redoReqCmd = m.quotaDisplay.RedoRequest()
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
	}
	var updateCmd tea.Cmd
	m.nextChoices, updateCmd = m.nextChoices.Update(msg)
	return m, tea.Batch(redoReqCmd, reqCmd, updateCmd)
}

func (m storageModel) View() string {
	if m.quotaDisplay.Response == nil {
		return m.quotaDisplay.View()
	}
	return fmt.Sprintf(
		"%s\n%s\n",
		m.quotaDisplay.View(),
		m.nextChoices.View(),
	)
}
