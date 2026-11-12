package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type session struct {
	Name   string `json:"name"`
	Agent  string `json:"agent"`
	Status string `json:"status"`
	Live   bool   `json:"live"`
}

type model struct {
	port     int
	sessions []session
	agents   []string
	cursor   int
	err      error
	status   string
}

type tickMsg time.Time
type dataMsg struct {
	sessions []session
	agents   []string
	err      error
}

func New(port int) tea.Model {
	return model{port: port, status: "loading..."}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(fetchCmd(m.port), tickCmd())
}

func tickCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func fetchCmd(port int) tea.Cmd {
	return func() tea.Msg {
		msg := dataMsg{}
		// sessions
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/sessions", port))
		if err != nil {
			msg.err = err
			return msg
		}
		defer resp.Body.Close()
		if err := json.NewDecoder(resp.Body).Decode(&msg.sessions); err != nil {
			msg.err = err
			return msg
		}
		// agents
		resp2, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/agents", port))
		if err == nil {
			defer resp2.Body.Close()
			var as []struct {
				Name string `json:"name"`
			}
			if json.NewDecoder(resp2.Body).Decode(&as) == nil {
				for _, a := range as {
					msg.agents = append(msg.agents, a.Name)
				}
			}
		}
		return msg
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			return m, fetchCmd(m.port)
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.sessions)-1 {
				m.cursor++
			}
		case "l":
			// launch default first agent
			if len(m.agents) > 0 {
				agent := m.agents[0]
				go launch(m.port, agent, "")
				m.status = "launched " + agent
			}
		case "x":
			if len(m.sessions) > 0 {
				name := m.sessions[m.cursor].Name
				go kill(m.port, name)
				m.status = "killed " + name
			}
		}
	case tickMsg:
		return m, tea.Batch(fetchCmd(m.port), tickCmd())
	case dataMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.sessions = msg.sessions
			if len(msg.agents) > 0 {
				m.agents = msg.agents
			}
			m.err = nil
		}
	}
	return m, nil
}

func launch(port int, agent, name string) {
	body := fmt.Sprintf(`{"agent":%q,"name":%q}`, agent, name)
	_, _ = http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/launch", port), "application/json", bytes.NewBufferString(body))
}

func kill(port int, name string) {
	body := fmt.Sprintf(`{"name":%q}`, name)
	_, _ = http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/kill", port), "application/json", bytes.NewBufferString(body))
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("error: %v\n(q to quit)", m.err)
	}
	title := lipgloss.NewStyle().Bold(true).Render("Cockpit TUI") + "\n"
	help := lipgloss.NewStyle().Faint(true).Render("j/k move  l launch  x kill  r refresh  q quit") + "\n\n"
	if len(m.sessions) == 0 {
		return title + help + "no sessions\n" + m.status
	}
	var lines string
	for i, s := range m.sessions {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		live := "live"
		if !s.Live {
			live = "dead"
		}
		lines += fmt.Sprintf("%s%-20s %-10s %-10s %s\n", cursor, s.Name, s.Agent, s.Status, live)
	}
	return title + help + lines + "\n" + m.status
}
