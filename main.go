package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	game Game
}

func initialModel(pch <-chan []Player, cch chan<- Command) model {
	return model{
		Game{
			field_x:        50,
			field_y:        30,
			emptyFiledRune: '.',
			players:        []Player{},
			commandsChan:   cch,
			playersChan:    pch,
		},
	}
}
func (m model) Init() tea.Cmd {
	return receiveState(m.game.playersChan)
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case []Player:
		m.game.players = msg
		return m, receiveState(m.game.playersChan)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			m.game.MoveMe(UP)
			return m, nil
		case "down", "j":
			m.game.MoveMe(DOWN)
			return m, nil
		case "left", "h":
			m.game.MoveMe(LEFT)
			return m, nil
		case "right", "l":
			m.game.MoveMe(RIGHT)
			return m, nil
		}
	}
	return m, nil
}

func (m model) View() string {
	return m.game.Render()
}

func receiveState(pch <-chan []Player) tea.Cmd {
	return func() tea.Msg {
		return <-pch
	}
}

func main() {
	playersChan, commandsChan := runMockServer()
	p := tea.NewProgram(initialModel(playersChan, commandsChan))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
