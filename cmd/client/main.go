package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"time"
)

type model struct {
	game LocalGame
}

type Connection struct {
	gameStateChan <-chan GameState
	commandsChan  chan<- Command
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
	case GameState:
		m.game.currentState = msg
		return m, receiveState(m.game.connection.gameStateChan)
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

func receiveState(pch <-chan GameState) tea.Cmd {
	return func() tea.Msg {
		return <-pch
	}
}

type Command = int

const (
	UP = iota
	DOWN
	LEFT
	RIGHT
)

type Player struct {
	playerRune rune
	x          int
	y          int
}

type GameState struct {
	players []Player
}

type LocalGame struct {
	field_x        int
	field_y        int
	emptyFiledRune rune
	currentState   GameState
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

	for _, p := range g.currentState.players {
		field[p.y][p.x] = p.playerRune
	}

	res := ""
	for _, row := range field {
		res += string(row) + "\n"
	}
	return res
}

func (g *LocalGame) MoveMe(direction Command) {
	g.connection.commandsChan <- direction
}

// "Mocked" runMockServer from which we receive game state,
// and to which we send commands
func runMockServer() Connection {
	gsch := make(chan GameState)
	cch := make(chan Command)

	players := []Player{
		{x: 3, y: 5, playerRune: 'K'},
		{x: 14, y: 27, playerRune: 'S'},
	}

	go func() {
		mockBotMovementTicker := time.NewTicker(time.Second)
		for {
			select {
			case <-mockBotMovementTicker.C:
				players[1].y--
				gsch <- GameState{players}
			case direction := <-cch:
				switch direction {
				case UP:
					players[0].y--
				case DOWN:
					players[0].y++
				case LEFT:
					players[0].x--
				case RIGHT:
					players[0].x++
				}
				gsch <- GameState{players}
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
