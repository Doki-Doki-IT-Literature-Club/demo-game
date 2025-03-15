package main

import (
	"fmt"
	types "github.com/Doki-Doki-IT-Literature-Club/demo-game/pkg/types"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"time"
)

type model struct {
	game LocalGame
}

type Connection struct {
	gameStateChan <-chan types.GameState
	commandsChan  chan<- types.Command
}

func initialModel(conn Connection) model {
	return model{
		LocalGame{
			field_x:        50,
			field_y:        30,
			emptyFiledRune: '.',
			connection:     conn,
		},
	}
}

func (m model) Init() tea.Cmd {
	return receiveState(m.game.connection.gameStateChan)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case types.GameState:
		m.game.currentState = msg
		return m, receiveState(m.game.connection.gameStateChan)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			m.game.MoveMe(types.UP)
			return m, nil
		case "down", "j":
			m.game.MoveMe(types.DOWN)
			return m, nil
		case "left", "h":
			m.game.MoveMe(types.LEFT)
			return m, nil
		case "right", "l":
			m.game.MoveMe(types.RIGHT)
			return m, nil
		}
	}
	return m, nil
}

func (m model) View() string {
	return m.game.Render()
}

func receiveState(pch <-chan types.GameState) tea.Cmd {
	return func() tea.Msg {
		return <-pch
	}
}

type LocalGame struct {
	field_x        int
	field_y        int
	emptyFiledRune rune
	currentState   types.GameState
	connection     Connection
}

func (g *LocalGame) Render() string {
	field := [][]rune{}
	for range g.field_y {
		row := []rune{}
		for range g.field_x {
			row = append(row, g.emptyFiledRune)
		}
		field = append(field, row)
	}

	for _, p := range g.currentState.Players {
		field[p.Y][p.X] = p.PlayerRune
	}

	res := ""
	for _, row := range field {
		res += string(row) + "\n"
	}
	return res
}

func (g *LocalGame) MoveMe(direction types.Command) {
	g.connection.commandsChan <- direction
}

// "Mocked" runMockServer from which we receive game state,
// and to which we send commands
func runMockServer() Connection {
	gsch := make(chan types.GameState)
	cch := make(chan types.Command)

	players := []types.Player{
		{X: 3, Y: 5, PlayerRune: 'K'},
		{X: 14, Y: 27, PlayerRune: 'S'},
	}

	go func() {
		mockBotMovementTicker := time.NewTicker(time.Second)
		for {
			select {
			case <-mockBotMovementTicker.C:
				players[1].Y--
				gsch <- types.GameState{Players: players}
			case direction := <-cch:
				switch direction {
				case types.UP:
					players[0].Y--
				case types.DOWN:
					players[0].Y++
				case types.LEFT:
					players[0].X--
				case types.RIGHT:
					players[0].X++
				}
				gsch <- types.GameState{Players: players}
			}
		}
	}()

	return Connection{gsch, cch}
}

func debug(s string) {
	fmt.Printf("--->%s<---", s)
}

func main() {
	conn := runMockServer()
	p := tea.NewProgram(initialModel(conn))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
