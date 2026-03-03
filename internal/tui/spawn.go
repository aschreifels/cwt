package tui

import (
	"fmt"
	"strings"

	"github.com/aschreifels/cwt/internal/workspace"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#a78bfa"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280"))
	labelStyle   = lipgloss.NewStyle().Width(14)
)

type stepState struct {
	name   string
	status string
	done   bool
	err    error
}

type Model struct {
	name       string
	branch     string
	worktree   string
	ticket     string
	steps      []stepState
	spinner    spinner.Model
	done       bool
	finalErr   error
	updates    <-chan workspace.StepUpdate
	result     *workspace.SpawnResult
}

type stepMsg workspace.StepUpdate
type doneMsg struct {
	result *workspace.SpawnResult
	err    error
}

func NewSpawnModel(name, branch, worktree, ticket string, updates <-chan workspace.StepUpdate) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#f59e0b"))

	return Model{
		name:     name,
		branch:   branch,
		worktree: worktree,
		ticket:   ticket,
		spinner:  s,
		updates:  updates,
		steps:    []stepState{},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, waitForUpdate(m.updates))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case stepMsg:
		m.updateStep(workspace.StepUpdate(msg))
		return m, waitForUpdate(m.updates)

	case doneMsg:
		m.done = true
		m.result = msg.result
		m.finalErr = msg.err
		return m, tea.Quit
	}

	return m, nil
}

func (m *Model) updateStep(update workspace.StepUpdate) {
	for i, s := range m.steps {
		if s.name == update.Pane {
			m.steps[i].status = update.Status
			m.steps[i].done = update.Done
			m.steps[i].err = update.Err
			return
		}
	}
	m.steps = append(m.steps, stepState{
		name:   update.Pane,
		status: update.Status,
		done:   update.Done,
		err:    update.Err,
	})
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(titleStyle.Render(" cwt spawn"))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Name:"), m.name))
	b.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Branch:"), m.branch))
	b.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Worktree:"), m.worktree))
	if m.ticket != "" {
		b.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Ticket:"), m.ticket))
	}
	b.WriteString("\n")

	for _, step := range m.steps {
		var icon string
		switch {
		case step.err != nil:
			icon = errorStyle.Render("✗")
		case step.done:
			icon = successStyle.Render("✓")
		default:
			icon = m.spinner.View()
		}

		status := step.status
		if step.err != nil {
			status = errorStyle.Render(status)
		} else if step.done {
			status = successStyle.Render(status)
		} else {
			status = dimStyle.Render(status)
		}

		b.WriteString(fmt.Sprintf("  %s %s  %s\n", icon, labelStyle.Render(step.name), status))
	}

	if m.done {
		b.WriteString("\n")
		if m.finalErr != nil {
			b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %s\n", m.finalErr)))
		} else {
			b.WriteString(successStyle.Render("  Workspace ready ✓\n"))
		}
	}

	b.WriteString("\n")
	return b.String()
}

func (m Model) Result() *workspace.SpawnResult {
	return m.result
}

func (m Model) Err() error {
	return m.finalErr
}

func waitForUpdate(ch <-chan workspace.StepUpdate) tea.Cmd {
	return func() tea.Msg {
		update, ok := <-ch
		if !ok {
			return doneMsg{}
		}
		return stepMsg(update)
	}
}

func RunSpawn(name, branch, worktree, ticket string, updates <-chan workspace.StepUpdate, resultCh <-chan SpawnDone) error {
	model := NewSpawnModel(name, branch, worktree, ticket, updates)
	p := tea.NewProgram(model)

	go func() {
		res := <-resultCh
		p.Send(doneMsg{result: res.Result, err: res.Err})
	}()

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	m := finalModel.(Model)
	if m.finalErr != nil {
		return m.finalErr
	}
	return nil
}

type SpawnDone struct {
	Result *workspace.SpawnResult
	Err    error
}
