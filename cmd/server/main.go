package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	server "github.com/Doki-Doki-IT-Literature-Club/demo-game/pkg/server"
	"github.com/Doki-Doki-IT-Literature-Club/demo-game/pkg/types"
)

type model struct {
	logBuffer *strings.Builder
	ge        *server.GameEngine
}

func initialModel(ge *server.GameEngine, logBuffer *strings.Builder) model {
	m := model{logBuffer, ge}
	return m
}

type InterfaceUpdate = int

func interfaceTimer(logBuffer *strings.Builder) tea.Cmd {
	return func() tea.Msg {
		sleepDur := 20 * time.Millisecond
		t := time.NewTicker(sleepDur)
		for range t.C {
			if logBuffer.Len() > 0 {
				return InterfaceUpdate(1)
			}
		}
		return InterfaceUpdate(0)
	}
}

func (m model) Init() tea.Cmd {
	return interfaceTimer(m.logBuffer)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, nil

	case InterfaceUpdate:
		return m, interfaceTimer(m.logBuffer)
	case tea.KeyPressMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	return m, cmd
}

func getInterfaceString(gameState *types.GameState) string {
	playerInfo := []string{"Players:"}
	for _, player := range gameState.Players {
		playerInfo = append(playerInfo, fmt.Sprintf("ID: %v %v HP: %v",
			player.ID, player.Position.ToString(), player.HP))
	}
	return strings.Join(playerInfo, "\n")
}

func (m model) View() tea.View {
	serverInterface := getInterfaceString(&m.ge.State)
	serverInterface = fmt.Sprintf("%v\nLogs:\n%v", serverInterface, "AAA")
	return tea.NewView(serverInterface)
}

func main() {
	port := ""

	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	logBuffer := &strings.Builder{}
	ge := server.RunGameEngine(logBuffer)
	m := initialModel(ge, logBuffer)
	go server.RunServer(port, ge)
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
