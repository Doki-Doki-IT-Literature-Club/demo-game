package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"slices"

	types "github.com/Doki-Doki-IT-Literature-Club/demo-game/pkg/types"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	defaultServerAddress = "localhost:8000"
	mapObjRenderChar     = '#'
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
			field_x:        types.FieldMaxX,
			field_y:        types.FieldMaxY,
			emptyFiledRune: ' ',
			connection:     conn,
			mapObjects:     types.MapObjects,
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
			m.game.SendCommand(types.UP)
			return m, nil
		case "down", "j":
			m.game.SendCommand(types.DOWN)
			return m, nil
		case "left", "h":
			m.game.SendCommand(types.LEFT)
			return m, nil
		case "right", "l":
			m.game.SendCommand(types.RIGHT)
			return m, nil
		case "e":
			m.game.SendCommand(types.SHOOT)
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
	mapObjects     []types.MapObject
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
		field[int32(p.Position.Y)][int32(p.Position.X)] = p.PlayerRune
	}

	for _, p := range g.currentState.Projectiles {
		field[int32(p.Position.Y)][int32(p.Position.X)] = p.Rune
	}

	for _, mo := range g.mapObjects {
		if !mo.IsVisible {
			continue
		}
		for y := mo.BottmLeft.Y; y < mo.TopRight.Y; y++ {
			for x := mo.BottmLeft.X; x < mo.TopRight.X; x++ {
				// TODO: textures?
				field[int32(y)][int32(x)] = mapObjRenderChar
			}
		}
	}

	res := fmt.Sprintf("Players: %d, projectiles: %d\n", len(g.currentState.Players), len(g.currentState.Projectiles))
	slices.Reverse(field)
	for _, row := range field {
		res += string(row) + "\n"
	}
	return res
}

func (g *LocalGame) SendCommand(direction types.Command) {
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
			buff := make([]byte, 2, 2)
			_, err := io.ReadFull(conn, buff)
			if err == io.EOF {
				continue
			}
			if err != nil {
				panic(err)
			}
			playerNumber := int(buff[0])
			projectileNumber := int(buff[1])
			gameStateSize := playerNumber*16 + projectileNumber*12
			buff = make([]byte, gameStateSize, gameStateSize)
			_, err = io.ReadFull(conn, buff)
			if err != nil {
				panic(err)
			}
			gs := types.GameStateFromBytes(buff, playerNumber, projectileNumber)
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

func main() {
	serverAddress := defaultServerAddress
	if len(os.Args) > 1 {
		serverAddress = os.Args[1]
	}
	conn := connectToServer(serverAddress)
	p := tea.NewProgram(initialModel(conn))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
