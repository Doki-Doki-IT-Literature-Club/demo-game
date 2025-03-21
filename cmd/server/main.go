package main

import (
	"fmt"
	"io"
	"net"
	"slices"
	"sync"

	"github.com/Doki-Doki-IT-Literature-Club/demo-game/pkg/types"
)

type ClinetConn struct {
	write chan<- types.GameState
}

type engineCommand struct {
	playerID types.PlayerID
	command  types.Command
}

type GameEngine struct {
	newPlayerID types.PlayerID
	conns       []*ClinetConn
	state       types.GameState
	engineInput chan engineCommand

	mu sync.Mutex
}

func (ge *GameEngine) addPlayer(conn *ClinetConn) types.PlayerID {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	newID := ge.newPlayerID
	ge.newPlayerID++
	ge.conns = append(ge.conns, conn)
	ge.state.Players = append(
		ge.state.Players,
		types.Player{
			ID:         newID,
			PlayerRune: 'G',
			X:          uint32(len(ge.state.Players)),
			Y:          uint32(len(ge.state.Players)),
		},
	)
	return newID
}

func (ge *GameEngine) HangleConnection(conn net.Conn) {
	fmt.Printf("New connection: %v\n", conn)

	write := make(chan types.GameState)

	cliConn := &ClinetConn{write}

	playerID := ge.addPlayer(cliConn)

	go func() {
		for state := range write {
			_, err := conn.Write(state.ToBytes())
			if err != nil {
				panic(err)
			}
		}
	}()

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
			ge.engineInput <- engineCommand{command: types.Command(buff[0]), playerID: playerID}
		}
	}()
	// TODO: handle connection close
}

func (ge *GameEngine) applyCommand(cmd engineCommand) {
	playerIDX := slices.IndexFunc(ge.state.Players, func(p types.Player) bool { return p.ID == cmd.playerID })
	if playerIDX == -1 {
		return
	}
	switch cmd.command {
	case types.UP:
		ge.state.Players[playerIDX].Y--
	case types.DOWN:
		ge.state.Players[playerIDX].Y++
	case types.LEFT:
		ge.state.Players[playerIDX].X--
	case types.RIGHT:
		ge.state.Players[playerIDX].X++
	}
}

func (ge *GameEngine) Run() {
	for ec := range ge.engineInput {
		fmt.Printf("new command: %v", ec)
		ge.applyCommand(ec)
		for _, cli := range ge.conns {
			cli.write <- ge.state
		}
	}
}

func RunGameEngine() *GameEngine {
	ge := &GameEngine{
		state:       types.GameState{Players: []types.Player{}},
		conns:       []*ClinetConn{},
		engineInput: make(chan engineCommand),
	}
	go ge.Run()
	return ge
}

func main() {

	fmt.Println("starting server")

	ge := RunGameEngine()

	listner, err := net.Listen("tcp", "0.0.0.0:8000")
	if err != nil {
		panic(err)
	}

	for {
		conn, err := listner.Accept()
		if err != nil {
			panic(err)
		}

		go ge.HangleConnection(conn)
	}
}
