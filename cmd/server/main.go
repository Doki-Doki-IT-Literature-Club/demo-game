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
	gameTick           = 20 * time.Millisecond
	defaultPort        = 8000
	XSLOW              = 0.2
	YSLOW              = 0.2
	MAX_X_SPEED        = 2.3
	FRICTION_BOUNDARY  = 0.7
	PLAYER_X_SPEED_INC = 2.3
	PLAYER_Y_SPEED_INC = 4
	GRAVITY_SPEED_INC  = 0.7
)

type ClinetConn struct {
	write chan<- types.GameState
}

type engineCommand struct {
	playerID types.ObjectID
	command  types.Command
}

type GameEngine struct {
	newPlayerID     types.ObjectID
	newProjectileID types.ObjectID
	playerCommands  []engineCommand

	conns       map[types.ObjectID]*ClinetConn
	state       types.GameState
	engineInput chan engineCommand

	mu sync.Mutex
}

func (ge *GameEngine) addPlayer(conn *ClinetConn) types.ObjectID {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	newID := ge.newPlayerID
	ge.newPlayerID++
	ge.conns[newID] = conn
	ge.state.Players[newID] = &types.Player{
		ID:            newID,
		ViewDirection: types.D_RIGHT,
		Position: types.Vector{
			X: float64(len(ge.state.Players)),
			Y: float64(len(ge.state.Players)),
		},
		CollisionArea: types.CollisionArea{X: 0.9, Y: 0.9},
		HP:            5,
	}
	return newID
}

func (ge *GameEngine) AddProjectile(position types.Vector, speed types.Vector) {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	newID := ge.newProjectileID
	ge.newProjectileID++
	ge.state.Projectiles[newID] = &types.Projectile{
		ID:            newID,
		Rune:          '•',
		Position:      position,
		Speed:         speed,
		CollisionArea: types.CollisionArea{X: 1, Y: 1},
	}
}

func (ge *GameEngine) removeProjectile(projectileID types.ObjectID) {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	delete(ge.state.Projectiles, projectileID)
}

func (ge *GameEngine) disconnectPlayer(playerID types.ObjectID) {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	delete(ge.state.Players, playerID)
	delete(ge.conns, playerID)

}

func (ge *GameEngine) HandleConnection(conn net.Conn) {
	//fmt.Printf("New connection: %v\n", conn)
	write := make(chan types.GameState)
	cliConn := &ClinetConn{write}
	playerID := ge.addPlayer(cliConn)
	initData := types.InitializationData{PlayerID: playerID}
	_, err := conn.Write(initData.ToBytes())
	if err != nil {
		ge.disconnectPlayer(playerID)
		return
	}

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
			buff := make([]byte, 1)
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

func (ge *GameEngine) detectCollision(
	selfID types.ObjectID,
	currentBox types.CollisionBox,
	movement types.Vector,
) types.CollidableObject {
	possibleCollisionBox := currentBox.Add(movement)

	//fmt.Printf("Possible collision box: %s\n", possibleCollisionBox.ToString())
	//fmt.Printf("Last possible collision box: %s\n", currentBox.ToString())
	//fmt.Printf("Movment vector: %+v\n", movement.ToString())

	for _, mo := range ge.state.MapObjects {
		moCollisionBox := mo.CollisionArea.ToCollisionBox(mo.Position)
		if moCollisionBox.IntersectsWith(possibleCollisionBox) {
			return &mo
		}
	}

	for _, p := range ge.state.Players {
		if p.ID == selfID {
			continue
		}
		playerCollisionBox := p.CollisionArea.ToCollisionBox(p.Position)
		if playerCollisionBox.IntersectsWith(possibleCollisionBox) {
			return p
		}
	}
	return nil
}

func getSpeedsAfterCollision(
	current types.CollisionBox,
	speed types.Vector,
	stepVector types.Vector,
	collidesWith types.CollidableObject,
) (types.Vector, types.Vector) {
	collidedWithBox := collidesWith.GetCollisionArea().ToCollisionBox(collidesWith.GetPosition())
	if collidedWithBox.IntersectsWithX(current) {
		// Already was within X bounds, meaning collision happend during Y movement
		//fmt.Printf("* Y Collision detected with %s\n", collidedWithBox.ToString())
		speed.Y = 0
		stepVector.Y = 0
	} else if collidedWithBox.IntersectsWithY(current) {
		// Already was within Y bounds, meaning collision happend during X movement
		//fmt.Printf("* X Collision detected with %s\n", collidedWithBox.ToString())
		speed.X = 0
		stepVector.X = 0
	} else {
		// Diagonal collision
		speed.X = 0
		speed.Y = 0
		stepVector.X = 0
		stepVector.Y = 0
		//fmt.Printf("Diagonal collision with %s\n", collidedWithBox.ToString())
	}
	return speed, stepVector
}

func getStepVectorWithIterations(v types.Vector) (types.Vector, int) {
	singleVector := v.SingleVector()
	maxIterations := int32(math.Round(v.GetLen()))
	if maxIterations < 4 {
		singleVector = v.Multiply(0.1)
		maxIterations = maxIterations * 10
	}
	return singleVector, int(maxIterations)
}

func (ge *GameEngine) MoveObject(obj types.MovableObject) types.CollidableObject {
	speed := obj.GetSpeed()
	stepVector, maxIterations := getStepVectorWithIterations(speed)

	lastPossibleCollisionBox := obj.GetCollisionArea().ToCollisionBox(obj.GetPosition())

	var collidesWith types.CollidableObject = nil
	for range maxIterations {
		possibleCollision := ge.detectCollision(obj.GetID(), lastPossibleCollisionBox, stepVector)
		if possibleCollision == nil {
			lastPossibleCollisionBox = lastPossibleCollisionBox.Add(stepVector)
			continue
		}

		collidesWith = possibleCollision

		speed, stepVector = getSpeedsAfterCollision(
			lastPossibleCollisionBox,
			speed,
			stepVector,
			collidesWith,
		)

		possibleCollision = ge.detectCollision(obj.GetID(), lastPossibleCollisionBox, stepVector)
		if possibleCollision == nil {
			lastPossibleCollisionBox = lastPossibleCollisionBox.Add(stepVector)
			continue
		}
		collidesWith = possibleCollision

		speed, stepVector = getSpeedsAfterCollision(
			lastPossibleCollisionBox,
			speed,
			stepVector,
			collidesWith,
		)
	}
	obj.SetSpeed(speed)
	obj.SetPosition(lastPossibleCollisionBox.BottomLeft)
	return collidesWith
}

func (ge *GameEngine) calculateState() {
	for _, player := range ge.state.Players {
		// "gravity"
		player.Speed.Y -= GRAVITY_SPEED_INC

		//fmt.Printf("%s\n", player.ToString())

		ge.MoveObject(player)

		if player.HP == 0 {
			ge.disconnectPlayer(player.ID)
			continue
		}

		// "slowing"
		// TODO: airborn not working now
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
			slowY := math.Pow(player.Speed.Y, 2) * YSLOW
			newSpeed.Y = player.Speed.Y - slowY
		}
		if player.Speed.Y < 0 {
			slowY := math.Pow(player.Speed.Y, 2) * YSLOW
			newSpeed.Y = player.Speed.Y - -slowY
		}

		player.Speed = newSpeed
	}

	for _, proj := range ge.state.Projectiles {
		//fmt.Printf("Projectile %s\n", proj.Position.ToString())
		collidesWith := ge.MoveObject(proj)
		if collidesWith != nil {
			//fmt.Printf("Collides with: %v\n", collidesWith)
			collidesWith.OnCollision(proj)
			ge.removeProjectile(proj.ID)
		}
	}
}

func (ge *GameEngine) saveCommand(cmd engineCommand) {
	ge.mu.Lock()
	defer ge.mu.Unlock()
	ge.playerCommands = append(ge.playerCommands, cmd)
}

func (ge *GameEngine) applyCommands() {
	ge.mu.Lock()
	defer ge.mu.Unlock()
	for _, c := range ge.playerCommands {
		ge.applyCommand(c)
	}
	clear(ge.playerCommands)
}

func (ge *GameEngine) applyCommand(cmd engineCommand) {
	player, ok := ge.state.Players[cmd.playerID]
	if !ok {
		return
	}
	switch cmd.command {
	case types.UP:
		player.Speed = player.Speed.Add(types.Vector{X: 0, Y: PLAYER_Y_SPEED_INC})
		// TODO: update player direction, don't set Rune
		player.ViewDirection = types.D_UP
	case types.DOWN:
		player.Speed = player.Speed.Add(types.Vector{X: 0, Y: -PLAYER_Y_SPEED_INC})
		player.ViewDirection = types.D_DOWN
	case types.LEFT:
		if player.Speed.X < -MAX_X_SPEED {
			break
		}
		player.Speed = player.Speed.Add(types.Vector{X: -PLAYER_X_SPEED_INC, Y: 0})
		player.ViewDirection = types.D_LEFT
	case types.RIGHT:
		if player.Speed.X > MAX_X_SPEED {
			break
		}
		player.Speed = player.Speed.Add(types.Vector{X: PLAYER_X_SPEED_INC, Y: 0})
		player.ViewDirection = types.D_RIGHT
	case types.SHOOT:
		ge.AddProjectile(
			player.Position.Add(player.ViewDirection.AsVector()),
			player.ViewDirection.AsVector().Multiply(2.0),
		)
	}
}

func (ge *GameEngine) Run() {
	ticker := time.NewTicker(gameTick)
	go func() {
		for ec := range ge.engineInput {
			//fmt.Printf("new command: %+v\n", ec)
			ge.saveCommand(ec)
		}
	}()

	t := time.Now()
	for range ticker.C {
		fmt.Printf("elapsed: %d\n", time.Since(t).Milliseconds())
		ge.applyCommands()
		ge.calculateState()
		for _, cli := range ge.conns {
			cli.write <- ge.state
		}
		t = time.Now()
	}
}

func RunGameEngine() *GameEngine {
	ge := &GameEngine{
		state: types.GameState{
			Players:     types.PlayerMap{},
			Projectiles: types.ProjectileMap{},
			MapObjects:  types.MapObjects,
		},
		conns:       map[types.ObjectID]*ClinetConn{},
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

	//fmt.Printf("starting server on %s\n", addr)

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

		go ge.HandleConnection(conn)
	}
}
