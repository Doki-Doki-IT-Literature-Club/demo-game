package main

import (
	"fmt"
	"net"
	"os"
	"slices"
	"time"

	tea "charm.land/bubbletea/v2"
	types "github.com/Doki-Doki-IT-Literature-Club/demo-game/pkg/types"
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

func initialModel(conn Connection, initData types.InitializationData) model {
	return model{
		game: LocalGame{
			field_x:        types.FieldMaxX,
			field_y:        types.FieldMaxY,
			emptyFiledRune: ' ',
			playerID:       initData.PlayerID,
			connection:     conn,
			mapObjects:     types.MapObjects,
			keysPressed:    0,
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
		case "shift+right", "L":
			m.game.SendCommand(types.RIGHT_RUN)
		case "shift+left", "H":
			m.game.SendCommand(types.LEFT_RUN)
		case "up", "k", "shift+up", "K":
			m.game.SendCommand(types.UP)
			return m, nil
		case "down", "j", "shift+down", "J":
			m.game.SendCommand(types.DOWN)
			return m, nil
		case "left", "h":
			m.game.SendCommand(types.LEFT)
			return m, nil
		case "right", "l":
			m.game.SendCommand(types.RIGHT)
			return m, nil
		case "e":
			m.game.keysPressed++
			m.game.SendCommand(types.SHOOT)
			return m, nil
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	return tea.NewView(m.game.Render())
}

func receiveState(gamestateChan <-chan types.GameState) tea.Cmd {
	return func() tea.Msg {
		sleepDur := 20 * time.Millisecond
		var state types.GameState
		t := time.NewTicker(sleepDur)
		gotState := false

		for {
			select {
			case s, ok := <-gamestateChan:
				if ok {
					state = s
					gotState = true
				}
			case <-t.C:
				if gotState {
					return state
				}
			}
		}
	}
}

var prevRender time.Time = time.Now()
var maxrenderms int64 = 0

type LocalGame struct {
	field_x        int
	field_y        int
	emptyFiledRune rune
	currentState   types.GameState
	connection     Connection
	playerID       types.ObjectID
	mapObjects     []types.MapObject
	keysPressed    int
}

func (g *LocalGame) getInterfaceRow() string {
	elapsed := time.Since(prevRender).Milliseconds()
	maxrenderms = max(maxrenderms, elapsed)
	debugInfo := fmt.Sprintf(
		"[Tick: %d] Players: %d, projectiles: %d, time from prev render (ms): %d, max render time (ms): %d\n",
		g.currentState.TickNumber,
		len(g.currentState.Players),
		len(g.currentState.Projectiles),
		elapsed,
		maxrenderms,
	)
	prevRender = time.Now()
	interfaceString := "INTERFACE HERE\n"
	localPlyaer, exists := g.currentState.Players[g.playerID]
	if exists {
		interfaceString = fmt.Sprintf("HP: %d, Coords: %s, Keys pressed: %d\n",
			localPlyaer.HP,
			localPlyaer.Position.ToString(),
			g.keysPressed,
		)
	}
	return fmt.Sprintf("%s%s", debugInfo, interfaceString)
}

func (g *LocalGame) Render() string {
	field := make([][]rune, g.field_y)
	row := make([]rune, g.field_x)
	for x := range g.field_x {
		row[x] = g.emptyFiledRune
	}
	for y := range g.field_y {
		field[y] = slices.Clone(row)
	}

	for _, p := range g.currentState.Players {
		x := int(p.Position.X)
		y := int(p.Position.Y)
		if x < 0 || x > g.field_x || y < 0 || y > g.field_y {
			continue
		}
		field[y][x] = p.ViewDirection.AsRune()
	}

	for _, p := range g.currentState.Projectiles {
		x := int(p.Position.X)
		y := int(p.Position.Y)
		if x < 0 || x > g.field_x || y < 0 || y > g.field_y {
			continue
		}
		field[y][x] = p.Rune
	}

	for _, mo := range g.mapObjects {
		if !mo.IsVisible {
			continue
		}
		cb := mo.GetCollisionBox()
		for y := cb.BottomLeft.Y; y < cb.TopRight.Y; y++ {
			for x := cb.BottomLeft.X; x < cb.TopRight.X; x++ {
				// TODO: textures?
				field[int32(y)][int32(x)] = mapObjRenderChar
			}
		}
	}

	res := ""
	slices.Reverse(field)
	for _, row := range field {
		res += string(row) + "\n"
	}
	res += g.getInterfaceRow()
	return res
}

func (g *LocalGame) SendCommand(direction types.Command) {
	g.connection.commandsChan <- direction
}

func connectToServer(serverAddress string) (Connection, types.InitializationData) {
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		fmt.Println("Error connecting:", err)
		os.Exit(1)
	}

	fmt.Println("Connected to", serverAddress)
	initializationData := types.InitializationDataFromBytes(conn)

	gameStateChannel := make(chan types.GameState, 10)
	go func() {
		for {
			gs := types.GameStateFromBytes(conn)
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

	return Connection{gameStateChannel, commandChannel}, initializationData
}

func main() {
	serverAddress := defaultServerAddress
	if len(os.Args) > 1 {
		serverAddress = os.Args[1]
	}
	conn, initData := connectToServer(serverAddress)
	p := tea.NewProgram(initialModel(conn, initData))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
