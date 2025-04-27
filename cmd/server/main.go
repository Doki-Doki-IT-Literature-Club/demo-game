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
	gameTick          = 100 * time.Millisecond
	defaultPort       = 8000
	XSLOW             = 0.1
	YSLOW             = 0.1
	MAX_X_SPEED       = 7
	FRICTION_BOUNDARY = 0.7
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

func (ge *GameEngine) AddProjectile(projectile *types.Projectile) {
	ge.state.Projectiles = append(ge.state.Projectiles, projectile)
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

func (ge *GameEngine) MoveObject(p types.MovableObject) {
	speed := p.GetSpeed()
	position := p.GetPosition()

	if speed.X == 0 && speed.Y == 0 {
		return
	}

	singleVector := speed.SingleVector()
	maxIterations := int32(math.Round(speed.GetLen()))
	if maxIterations < 1 {
		singleVector = speed
		maxIterations = 1
	}

	lastPossible := types.Vector{X: position.X, Y: position.Y}
	fmt.Printf("single vector: %+v\n", singleVector.ToString())
	fmt.Printf("current position: %+v\n", lastPossible.ToString())

	// movementLoop:
	for range maxIterations {
		possiblePosition := lastPossible.Add(singleVector)
		fmt.Printf("possible position: %s\n", possiblePosition.ToString())

		collides := true

		i := 0
		for collides {
			collides = false
			for _, mo := range ge.state.MapObjects {
				if mo.CollidesWith(possiblePosition) {
					collides = true
					if mo.IsWithinX(lastPossible) {
						// Already was within X bounds, meaning collision happend during Y movement
						fmt.Printf("* Y Collision detected with %s, %s\n", mo.BottmLeft.ToString(), mo.TopRight.ToString())
						speed.Y = 0
						singleVector.Y = 0
					} else if mo.IsWithinY(lastPossible) {
						// Already was within Y bounds, meaning collision happend during X movement
						fmt.Printf("* X Collision detected with %s, %s\n", mo.BottmLeft.ToString(), mo.TopRight.ToString())
						speed.X = 0
						singleVector.X = 0
					} else {
						// Diagonal collision
						speed.X = 0
						speed.Y = 0
						singleVector.X = 0
						singleVector.Y = 0
						fmt.Printf("Diagonal collision with %s, %s", mo.BottmLeft.ToString(), mo.TopRight.ToString())
					}
				}
			}
			possiblePosition = lastPossible.Add(singleVector)
			i += 1
			if i > 100 {
				panic("panic")
			}
			fmt.Printf("adjusted possible position: %s\n", possiblePosition.ToString())
		}
		lastPossible = possiblePosition
	}
	fmt.Printf("selected position: %s\n\n\n", lastPossible.ToString())
	p.SetSpeed(speed)
	p.SetPosition(lastPossible)
}

func (ge *GameEngine) calculateState() {
	for _, player := range ge.state.Players {
		// "gravity"
		player.Speed.Y -= 2

		fmt.Printf("%s\n", player.ToString())
		ge.MoveObject(player)

		// "slowing"
		newSpeed := types.Vector{}
		if player.Speed.X > 0 {
			slowX := math.Pow(player.Speed.X, 2) * XSLOW
			newSpeed.X = player.Speed.X - slowX
			if player.Speed.X < FRICTION_BOUNDARY && !player.IsAirborn {
				newSpeed.X = 0
			}
		}
		if player.Speed.X < 0 {
			slowX := math.Pow(player.Speed.X, 2) * XSLOW
			newSpeed.X = player.Speed.X - -slowX
			if player.Speed.X > -FRICTION_BOUNDARY && !player.IsAirborn {
				newSpeed.X = 0
			}
		}
		if player.Speed.Y > 0 {
			slowY := math.Pow(player.Speed.X, 2) * YSLOW
			newSpeed.Y = player.Speed.Y - slowY
		}
		if player.Speed.Y < 0 {
			slowY := math.Pow(player.Speed.X, 2) * YSLOW
			newSpeed.Y = player.Speed.Y - -slowY
		}

		player.Speed = newSpeed
	}

	for _, proj := range ge.state.Projectiles {
		fmt.Printf("Projectile %s\n", proj.Position.ToString())
		ge.MoveObject(proj)
	}
}

func (ge *GameEngine) applyCommand(cmd engineCommand) {
	player, ok := ge.state.Players[cmd.playerID]
	if !ok {
		return
	}
	switch cmd.command {
	case types.UP:
		player.Speed = player.Speed.Add(types.Vector{X: 0, Y: 5})
		// TODO: update player direction, don't set Rune
		player.PlayerRune = types.DirectionCharMap[cmd.command]
	case types.DOWN:
		player.Speed = player.Speed.Add(types.Vector{X: 0, Y: -5})
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
	case types.SHOOT:
		ge.AddProjectile(
			&types.Projectile{
				Position: player.Position.Add(types.Vector{X: 1, Y: 0}),
				Speed:    types.Vector{X: 1, Y: 0},
				Rune:     '•',
			},
		)
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
		state:       types.GameState{Players: map[types.PlayerID]*types.Player{}, MapObjects: types.MapObjects},
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
