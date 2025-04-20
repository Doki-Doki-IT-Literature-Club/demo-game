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
	XSLOW       = 0.1
	YSLOW       = 0.1
	MAX_X_SPEED = 7
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
		Position: types.Vector{
			X: float64(len(ge.state.Players)),
			Y: float64(len(ge.state.Players)),
		},
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

func (ge *GameEngine) MovePlayer(playerID types.PlayerID) {
	p := ge.state.Players[playerID]

	singleVector := p.Speed.SingleVector()
	maxIterations := int32(math.Round(p.Speed.GetLen()))
	if maxIterations < 1 {
		singleVector = p.Speed
		maxIterations = 1
	}

	lastPossible := types.Vector{X: p.Position.X, Y: p.Position.Y}
	fmt.Printf("single vector: %+v\n", singleVector.ToString())
	fmt.Printf("current position: %+v\n", lastPossible.ToString())

movementLoop:
	for i := range maxIterations {
		possibleMovement := singleVector.Multiply(float64(i + 1))
		possiblePosition := p.Position.Add(possibleMovement)
		fmt.Printf("Possible position: %s\n", possiblePosition.ToString())

		if possiblePosition.X >= types.FieldMaxX ||
			possiblePosition.Y >= types.FieldMaxY ||
			possiblePosition.X < 0 ||
			possiblePosition.Y < 0 {

			fmt.Printf("Break because of out of borders: %s\n", possiblePosition.ToString())
			if possiblePosition.X >= types.FieldMaxX-1 {
				possiblePosition.X = types.FieldMaxX - 1
				p.Speed.X = 0
			}
			if possiblePosition.X < 0 {
				possiblePosition.X = 0.0
				p.Speed.X = 0
			}

			if possiblePosition.Y >= types.FieldMaxY-1 {
				possiblePosition.Y = types.FieldMaxY - 1
				p.Speed.Y = 0
			}
			if possiblePosition.Y < 0 {
				possiblePosition.Y = 0.0
				p.Speed.Y = 0
			}
			lastPossible.X = possiblePosition.X
			lastPossible.Y = possiblePosition.Y
			break movementLoop
		}

		for pid, player := range ge.state.Players {
			if pid == p.ID {
				continue
			}
			if possiblePosition == player.Position {
				break movementLoop
			}
		}
		lastPossible = possiblePosition
	}
	fmt.Printf("selected position: %s\n\n\n", lastPossible.ToString())
	p.Position = lastPossible

}

func (ge *GameEngine) calculateState() {
	for pid, player := range ge.state.Players {
		ge.MovePlayer(pid)
		fmt.Printf("%s\n", player.ToString())

		// "slowing"
		newSpeed := types.Vector{}
		if player.Speed.X > 0 {
			slowX := math.Pow(player.Speed.X, 2) * XSLOW
			newSpeed.X = player.Speed.X - slowX
		}
		if player.Speed.X < 0 {
			slowX := math.Pow(player.Speed.X, 2) * XSLOW
			newSpeed.X = player.Speed.X - -slowX
		}
		if player.Speed.Y > 0 {
			slowY := math.Pow(player.Speed.X, 2) * YSLOW
			newSpeed.Y = player.Speed.Y - slowY
		}
		if player.Speed.Y < 0 {
			slowY := math.Pow(player.Speed.X, 2) * YSLOW
			newSpeed.Y = player.Speed.Y - -slowY
		}

		// "gravity"
		if player.Position.Y != types.FieldMaxY-1 {
			newSpeed.Y += 2
		}
		player.Speed = newSpeed
	}
}

func (ge *GameEngine) applyCommand(cmd engineCommand) {
	player, ok := ge.state.Players[cmd.playerID]
	if !ok {
		return
	}
	switch cmd.command {
	case types.UP:
		player.Speed = player.Speed.Add(types.Vector{X: 0, Y: -5})
		// TODO: update player direction, don't set Rune
		player.PlayerRune = types.DirectionCharMap[cmd.command]
	case types.DOWN:
		player.Speed = player.Speed.Add(types.Vector{X: 0, Y: 5})
		player.PlayerRune = types.DirectionCharMap[cmd.command]
	case types.LEFT:
		if player.Speed.X < -MAX_X_SPEED {
			break
		}
		player.Speed = player.Speed.Add(types.Vector{X: -3, Y: 0})
		player.PlayerRune = types.DirectionCharMap[cmd.command]
	case types.RIGHT:
		if player.Speed.X > MAX_X_SPEED {
			break
		}
		player.Speed = player.Speed.Add(types.Vector{X: 3, Y: 0})
		player.PlayerRune = types.DirectionCharMap[cmd.command]
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
