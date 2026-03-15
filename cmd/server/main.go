package main

import (
	"bufio"
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
	server "github.com/Doki-Doki-IT-Literature-Club/demo-game/pkg/server"
)

type model struct {
	logBuffer *bufio.ReadWriter
}

func initialModel() model {
	logBuffer := bufio.NewReadWriter(&bufio.Reader{}, &bufio.Writer{})
	m := model{logBuffer}
	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, nil

	case tea.KeyPressMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "a":
			m.logBuffer.WriteString(fmt.Sprintf("LogWriter: %v", &m.logBuffer))
			return m, nil

		case "enter":
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	return m, cmd
}

func (m model) View() tea.View {
	return tea.NewView(m.logBuffer.ReadString())
}

func main() {
	port := ""

	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	m := initialModel()
	go server.RunServer(port, &m.logBuffer)
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
