package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/NTUEEECluster/storaged"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type webRequestModel struct {
	RequestDesc string
	Response    *webRequestModelResponse
	Spinner     spinner.Model

	autoExit bool
	modelID  int
	url      string
	jsonBody string
}

type webRequestModelResponse struct {
	modelID    int
	StatusCode int
	Body       string
	Error      error
}

var lastID int64

func nextID() int {
	return int(atomic.AddInt64(&lastID, 1))
}

func NewWebRequestModel(requestDesc string, url string, reqBody any) webRequestModel {
	s := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("227"))),
	)
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return webRequestModel{
			RequestDesc: requestDesc,
			Response: &webRequestModelResponse{
				Error: fmt.Errorf("error marshalling body: %w", err),
			},
			Spinner: s,
		}
	}
	return webRequestModel{
		RequestDesc: requestDesc,
		Response:    nil,
		Spinner:     s,
		modelID:     nextID(),
		url:         url,
		jsonBody:    string(reqJSON),
	}
}

func (m webRequestModel) RedoRequest() (webRequestModel, tea.Cmd) {
	m.Response = nil
	m.modelID = nextID()
	return m, m.doRequest
}

// AutoExit returns a webRequestModel that automatically exits.
func (m webRequestModel) AutoExit() webRequestModel {
	m.autoExit = true
	return m
}

func (m webRequestModel) Init() tea.Cmd {
	return tea.Batch(m.doRequest, m.Spinner.Tick)
}

func (m webRequestModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var spinnerCmd tea.Cmd
	m.Spinner, spinnerCmd = m.Spinner.Update(msg)
	switch msg := msg.(type) {
	case webRequestModelResponse:
		if msg.modelID != m.modelID {
			break
		}
		m.Response = &msg
		if m.autoExit {
			return m, tea.Quit
		}
	}
	return m, spinnerCmd
}

func (m webRequestModel) View() string {
	if m.Response == nil {
		return m.Spinner.View() + " " + m.RequestDesc + "\n"
	}
	switch {
	case m.Response.StatusCode == 200:
		return OKStyle.Render(m.Response.Body)
	case m.Response.StatusCode != 0:
		return fmt.Sprintf(
			"%s\n\n%s\n",
			ErrorTitleStyle.Render(
				"❌ The server returned a "+
					strconv.Itoa(m.Response.StatusCode)+" "+
					http.StatusText(m.Response.StatusCode),
			),
			ErrorStyle.Render(m.Response.Body),
		)
	case m.Response.Error != nil:
		return fmt.Sprintf(
			"%s\n%s\n",
			ErrorTitleStyle.Render("❌ Failed to contact storage daemon: "+m.Response.Error.Error()),
			ErrorStyle.Render("Please try again later and contact the administrator if this persists."),
		)
	default:
		return fmt.Sprintf(
			"%s\n%s\n",
			ErrorTitleStyle.Render("❌ Failed to contact storage daemon: Unknown error"),
			ErrorStyle.Render("Please try again later and contact the administrator if this persists."),
		)
	}
}

func (m webRequestModel) doRequest() tea.Msg {
	mungeBody, err := storaged.Munge(string(m.jsonBody))
	if err != nil {
		return webRequestModelResponse{
			Error:   fmt.Errorf("error creating signed request: %w", err),
			modelID: m.modelID,
		}
	}
	req, err := http.NewRequest("POST", m.url, strings.NewReader(mungeBody))
	if err != nil {
		return webRequestModelResponse{
			Error:   fmt.Errorf("error creating request: %w", err),
			modelID: m.modelID,
		}
	}
	resp, err := http.DefaultClient.Do(req)
	if resp == nil {
		return webRequestModelResponse{
			Error:   err,
			modelID: m.modelID,
		}
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return webRequestModelResponse{
			StatusCode: resp.StatusCode,
			Error:      fmt.Errorf("error reading response from server: %w", err),
			modelID:    m.modelID,
		}
	}
	return webRequestModelResponse{
		Body:       string(b),
		StatusCode: resp.StatusCode,
		Error:      nil,
		modelID:    m.modelID,
	}
}
