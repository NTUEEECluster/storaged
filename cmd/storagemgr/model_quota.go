package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/NTUEEECluster/storaged"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type quotaModel struct {
	Inputs      []textinput.Model
	focus       int
	madeRequest bool
	webModel    webRequestModel
	helpModel   help.Model

	HostURL    string
	ActionDesc string
	IsDelete   bool
}

func NewQuotaModel(hostURL string, actionDesc string, isDelete bool) quotaModel {
	projectName := textinput.New()
	projectName.Width = 22
	projectName.CharLimit = 20
	projectName.Placeholder = "ExampleProj1"
	projectName.Validate = storaged.ValidateProjectName
	projectName.Focus()
	tier := textinput.New()
	tier.Width = 20
	tier.CharLimit = 15
	tier.Placeholder = "hdd"
	tier.Validate = validateTierName
	size := textinput.New()
	size.Width = 5
	size.CharLimit = 5
	size.Placeholder = "10"
	size.Validate = validateSize
	if isDelete {
		size.SetValue("0")
	}
	return quotaModel{
		Inputs:      []textinput.Model{projectName, tier, size},
		focus:       0,
		madeRequest: false,
		helpModel:   help.New(),

		HostURL:    hostURL,
		ActionDesc: actionDesc,
		IsDelete:   isDelete,
	}
}

func (quotaModel) Init() tea.Cmd { return nil }

func (m quotaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.madeRequest {
		return m.updateResult(msg)
	}
	return m.updateForm(msg)
}

func (m quotaModel) View() string {
	if m.madeRequest {
		return fmt.Sprintf(
			"%s\n\n%s\n",
			m.webModel.View(),
			m.helpModel.ShortHelpView([]key.Binding{keybindContinue}),
		)
	}

	hasError := false
	statusDisplay := OKStyle.Render("Ready for submission.")
	for _, input := range m.Inputs {
		err := input.Validate(input.Value())
		if err != nil {
			hasError = true
			errMsg := err.Error()
			if len(errMsg) > 0 {
				errMsg = strings.ToUpper(errMsg[:1]) + errMsg[1:]
			}
			statusDisplay = ErrorStyle.Render(errMsg)
			break
		}
	}

	keybinds := []key.Binding{keybindPrev, keybindNext, keybindSubmit, keybindCancel}
	if hasError {
		keybinds = []key.Binding{keybindPrev, keybindNext, keybindCancel}
	}

	return fmt.Sprintf(
		"%s\n"+
			"%s\n"+
			"\n"+
			"%s  %s\n"+
			"%s  %s\n"+
			"\n"+
			"%s\n"+
			"%s\n",
		InputHeaderStyle.Render("Folder Name to "+m.ActionDesc),
		m.Inputs[0].View(),
		InputHeaderStyle.Width(20).Render("Storage Tier"), InputHeaderStyle.Render("New Folder Size (GB)"),
		m.Inputs[1].View(), m.Inputs[2].View(),
		statusDisplay,
		m.helpModel.ShortHelpView(keybinds),
	)
}

func (m quotaModel) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		validInputCount := 3
		if m.IsDelete {
			validInputCount = 2
		}
		for i := range m.Inputs {
			m.Inputs[i].Blur()
		}
		switch {
		case key.Matches(msg, keybindCancel):
			return m, ReturnToList
		case key.Matches(msg, keybindPrev):
			m.focus += validInputCount - 1 // Equivalent to -1.
			m.focus %= validInputCount
		case key.Matches(msg, keybindNext):
			m.focus++
			m.focus %= validInputCount
		case key.Matches(msg, keybindSubmit):
			hasErr := false
			for i, input := range m.Inputs {
				if input.Validate(input.Value()) != nil {
					m.focus = i
					hasErr = true
					break
				}
			}
			if hasErr {
				break
			}
			folderName := m.Inputs[0].Value()
			tierName := m.Inputs[1].Value()
			sizeInGB, err := strconv.Atoi(m.Inputs[2].Value())
			if err != nil {
				panic("unexpected error in size when validated: " + err.Error())
			}
			m.webModel = newUpdateRequest(m.HostURL, folderName, tierName, sizeInGB)
			m.madeRequest = true
			return m, m.webModel.Init()
		}
		m.Inputs[m.focus].Focus()
	}
	cmds := make([]tea.Cmd, len(m.Inputs))
	for i := range m.Inputs {
		m.Inputs[i], cmds[i] = m.Inputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

func (m quotaModel) updateResult(msg tea.Msg) (tea.Model, tea.Cmd) {
	webModel, cmd := m.webModel.Update(msg)
	m.webModel = webModel.(webRequestModel)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, keybindContinue) {
			return m, tea.Batch(cmd, ReturnToList)
		}
	}
	return m, cmd
}

func validateTierName(tier string) error {
	if strings.TrimSpace(tier) == "" {
		return errors.New("tier name is required")
	}
	return nil
}

func validateSize(size string) error {
	if strings.TrimSpace(size) == "" {
		return errors.New("size is required")
	}
	v, err := strconv.Atoi(size)
	if err != nil {
		return errors.New("size must be a valid number (do not include unit)")
	}
	if v < 0 {
		return errors.New("size must be a positive number")
	}
	return nil
}
