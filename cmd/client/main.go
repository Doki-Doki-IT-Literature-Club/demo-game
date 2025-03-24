package main

import (
	"fmt"
	types "github.com/Doki-Doki-IT-Literature-Club/demo-game/pkg/types"
	tea "github.com/charmbracelet/bubbletea"
	"io"
	"net"
	"os"
)

const SERVER_ADDRESS = "localhost:8000"

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
			field_x:        types.FieldMaxX,
			field_y:        types.FieldMaxY,
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

func connectToServer(serverAddress string) Connection {
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		fmt.Println("Error connecting:", err)
		os.Exit(1)
	}

	fmt.Println("Connected to", serverAddress)

	gameStateChannel := make(chan types.GameState)
	go func() {
		for {
			buff := make([]byte, 1, 1)
			_, err := io.ReadFull(conn, buff)
			if err == io.EOF {
				continue
			}
			if err != nil {
				panic(err)
			}
			gameStateSize := buff[0] * 16
			buff = make([]byte, gameStateSize, gameStateSize)
			_, err = io.ReadFull(conn, buff)
			if err != nil {
				panic(err)
			}
			gs := types.GameStateFromBytes(buff)
			gameStateChannel <- gs
		}
	}()

	commandChannel := make(chan types.Command)
	go func() {
		for cmd := range commandChannel {
			_, err := conn.Write([]byte{byte(cmd)})
			if err != nil {
				panic(err)
			}
		}
	}()

	return Connection{gameStateChannel, commandChannel}
}

func debug(s string) {
	fmt.Printf("--->%s<---", s)
}

func main() {
	conn := connectToServer(SERVER_ADDRESS)
	p := tea.NewProgram(initialModel(conn))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
