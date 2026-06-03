package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/oh-my-pi/omp/pkg/ai"
)

type sessionState int

const (
	stateReady sessionState = iota
	stateWaiting
	stateDone
)

type programModel struct {
	session  *Session
	provider ai.Provider
	viewport viewport.Model
	input    textinput.Model
	spinner  spinner.Model
	state    sessionState
	width    int
	height   int
}

type responseMsg string
type errorMsg error

func NewProgram(provider ai.Provider, req ai.Request) (*tea.Program, error) {
	sess, err := NewSession(provider, req)
	if err != nil {
		return nil, err
	}

	ti := textinput.New()
	ti.Placeholder = "Type your message..."
	ti.Focus()
	ti.CharLimit = 0
	ti.Width = 80

	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().PaddingLeft(1)

	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := programModel{
		session:  sess,
		provider: provider,
		viewport: vp,
		input:    ti,
		spinner:  s,
		state:    stateReady,
		width:    80,
		height:   24,
	}

	return tea.NewProgram(m, tea.WithAltScreen()), nil
}

func (m programModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

func (m programModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 2
		m.viewport.Height = msg.Height - 5
		m.input.Width = msg.Width - 4
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "enter":
			if m.state == stateReady {
				input := strings.TrimSpace(m.input.Value())
				if input == "" {
					return m, nil
				}
				m.input.SetValue("")
				m.state = stateWaiting
				return m, m.send(input)
			}
		}

	case responseMsg:
		m.state = stateReady
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil

	case errorMsg:
		m.state = stateReady
		m.session.AddMessage("system", fmt.Sprintf("Error: %v", msg))
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil
	}

	if m.state == stateWaiting {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m programModel) send(input string) tea.Cmd {
	return func() tea.Msg {
		resp, err := m.session.Send(context.Background(), input)
		if err != nil {
			return errorMsg(err)
		}
		return responseMsg(resp)
	}
}

func (m programModel) renderMessages() string {
	var b strings.Builder
	for _, msg := range m.session.Messages() {
		b.WriteString(msg.Render())
		b.WriteString("\n\n")
	}
	return b.String()
}

func (m programModel) View() string {
	if m.state == stateWaiting {
		return fmt.Sprintf(
			"%s\n\n%s\n%s\n\n%s",
			lipgloss.NewStyle().Bold(true).Render("🤖 omp — AI Coding Agent"),
			m.viewport.View(),
			m.spinner.View()+" Thinking...",
			m.input.View(),
		)
	}
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		lipgloss.NewStyle().Bold(true).Render("🤖 omp — AI Coding Agent"),
		m.viewport.View(),
		m.input.View(),
	)
}
