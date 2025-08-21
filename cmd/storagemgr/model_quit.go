package main

import tea "github.com/charmbracelet/bubbletea"

type quitModel struct{}

var _ tea.Model = (*quitModel)(nil)

func (quitModel) Init() tea.Cmd                       { return tea.Quit }
func (quitModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return nil, nil }
func (quitModel) View() string                        { return "" }
