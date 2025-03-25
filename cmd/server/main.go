package main

import (
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"sync"
	"time"

	"github.com/Doki-Doki-IT-Literature-Club/demo-game/pkg/types"
)

const (
	gameTick    = 100 * time.Millisecond
	defaultPort = 8000
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
	conns       map[types.PlayerID]*ClinetConn
	state       types.GameState
	engineInput chan engineCommand

	mu sync.Mutex
}

func (ge *GameEngine) addPlayer(conn *ClinetConn) types.PlayerID {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	newID := ge.newPlayerID
	ge.newPlayerID++
	ge.conns[newID] = conn
	ge.state.Players[newID] = &types.Player{
		ID:         newID,
		PlayerRune: 'G',
		X:          uint32(len(ge.state.Players)),
		Y:          uint32(len(ge.state.Players)),
	}
	return newID
}

func (ge *GameEngine) disconnectPlayer(playerID types.PlayerID) {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	delete(ge.state.Players, playerID)
	delete(ge.conns, playerID)

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
				ge.disconnectPlayer(playerID)
				return
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
				ge.disconnectPlayer(playerID)
				return
			}
			ge.engineInput <- engineCommand{command: types.Command(buff[0]), playerID: playerID}
		}
	}()
}

func (ge *GameEngine) calculateState() {
	for _, player := range ge.state.Players {
		player.ApplyVec()
		fmt.Printf("%+v\n", player)

		v := types.Vector{}
		if player.Vec.X != 0 {
			v.X = -player.Vec.X / int32(math.Abs(float64(player.Vec.X)))
		}
		if player.Vec.Y != 0 {
			v.Y = -player.Vec.Y / int32(math.Abs(float64(player.Vec.Y)))
		}

		// "gravity"
		if player.Y != types.FieldMaxY-1 {
			v.Y += 1
		}
		player.Vec.Add(v)
	}
}

func (ge *GameEngine) applyCommand(cmd engineCommand) {
	player, ok := ge.state.Players[cmd.playerID]
	if !ok {
		return
	}
	switch cmd.command {
	case types.UP:
		player.Vec.Add(types.Vector{X: 0, Y: -3})
	case types.DOWN:
		player.Vec.Add(types.Vector{X: 0, Y: 3})
	case types.LEFT:
		player.Vec.Add(types.Vector{X: -3, Y: 0})
	case types.RIGHT:
		player.Vec.Add(types.Vector{X: 3, Y: 0})
	}
}

func (ge *GameEngine) Run() {
	ticker := time.NewTicker(gameTick)
	for {
		select {
		case ec := <-ge.engineInput:
			fmt.Printf("new command: %+v\n", ec)
			ge.applyCommand(ec)
		case <-ticker.C:
			ge.calculateState()
			for _, cli := range ge.conns {
				cli.write <- ge.state
			}
		}
	}
}

func RunGameEngine() *GameEngine {
	ge := &GameEngine{
		state:       types.GameState{Players: map[types.PlayerID]*types.Player{}},
		conns:       map[types.PlayerID]*ClinetConn{},
		engineInput: make(chan engineCommand),
	}
	go ge.Run()
	return ge
}

func main() {
	port := fmt.Sprintf("%d", defaultPort)

	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	addr := "0.0.0.0:" + port

	fmt.Printf("starting server on %s\n", addr)

	ge := RunGameEngine()

	listner, err := net.Listen("tcp", addr)
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
