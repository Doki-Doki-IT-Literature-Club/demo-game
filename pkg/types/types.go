package types

import (
	"encoding/binary"
	"fmt"
	"math"
)

type PlayerID uint32

type Command byte

const (
	UP    = 0x01
	DOWN  = 0x02
	LEFT  = 0x03
	RIGHT = 0x04
	SHOOT = 0x05
)

var DirectionCharMap = map[Command]rune{
	UP:    '^',
	DOWN:  'V',
	LEFT:  '<',
	RIGHT: '>',
}

const FieldMaxX = 50
const FieldMaxY = 30

type Vector struct {
	X float64
	Y float64
}

func (v *Vector) Add(other Vector) Vector {
	return Vector{v.X + other.X, v.Y + other.Y}
}

func (v *Vector) Sub(other Vector) Vector {
	return Vector{v.X - other.X, v.Y - other.Y}
}

func (v *Vector) SingleVector() Vector {
	vectorLen := v.GetLen()
	if math.IsNaN(vectorLen) {
		return Vector{}
	}
	return Vector{v.X / vectorLen, v.Y / vectorLen}
}

func (v *Vector) GetLen() float64 {
	return math.Sqrt(math.Pow(v.X, 2) + math.Pow(v.Y, 2))
}

func (v *Vector) Multiply(a float64) Vector {
	return Vector{v.X * a, v.Y * a}
}

func (v *Vector) ToString() string {
	return fmt.Sprintf("{X: %.2f | Y: %.2f}", v.X, v.Y)
}

type Player struct {
	ID         PlayerID
	PlayerRune rune
	Position   Vector
	Speed      Vector
	IsAirborn  bool
}

func (p *Player) ToString() string {
	airbornStr := "[S]"
	if p.IsAirborn {
		airbornStr = "[A]"
	}
	return fmt.Sprintf("Player: %c%s, Position: %s, Speed: %s", p.PlayerRune, airbornStr, p.Position.ToString(), p.Speed.ToString())
}

type Projectile struct {
	Rune     rune
	Position Vector
	Speed    Vector
}

func (p *Projectile) ToString() string {
	return fmt.Sprintf("Projectile %c, Position: %s, Speed: %s", p.Rune, p.Position, p.Speed)
}

type GameState struct {
	Players     map[PlayerID]*Player
	Projectiles []*Projectile
	MapObjects  []MapObject
}

func (gs *GameState) ToBytes() []byte {
	res := []byte{byte(len(gs.Players)), byte(len(gs.Projectiles))}
	for _, p := range gs.Players {
		pb := [16]byte{}
		binary.BigEndian.PutUint32(pb[:4], uint32(p.ID))
		binary.BigEndian.PutUint32(pb[4:8], uint32(p.PlayerRune))
		binary.BigEndian.PutUint32(pb[8:12], uint32(p.Position.X))
		binary.BigEndian.PutUint32(pb[12:], uint32(p.Position.Y))
		res = append(res, pb[:]...)
	}
	for _, p := range gs.Projectiles {
		pb := [12]byte{}
		binary.BigEndian.PutUint32(pb[:4], uint32(p.Rune))
		binary.BigEndian.PutUint32(pb[4:8], uint32(p.Position.X))
		binary.BigEndian.PutUint32(pb[8:], uint32(p.Position.Y))
		res = append(res, pb[:]...)
	}
	return res
}

func GameStateFromBytes(data []byte, playerNumber int, projectileNumber int) GameState {
	gs := GameState{map[PlayerID]*Player{}, []*Projectile{}, MapObjects}
	for i := range playerNumber {
		k := i * 16

		playerID := PlayerID(binary.BigEndian.Uint32(data[k : k+4]))
		X := binary.BigEndian.Uint32(data[k+8 : k+12])
		Y := binary.BigEndian.Uint32(data[k+12 : k+16])
		p := Player{
			ID:         playerID,
			PlayerRune: rune(binary.BigEndian.Uint32(data[k+4 : k+8])),
			Position:   Vector{float64(X), float64(Y)},
		}
		gs.Players[playerID] = &p
	}

	for i := range projectileNumber {
		k := 16*playerNumber + i*12
		X := binary.BigEndian.Uint32(data[k+4 : k+8])
		Y := binary.BigEndian.Uint32(data[k+8 : k+12])
		p := Projectile{
			Rune:     rune(binary.BigEndian.Uint32(data[k : k+4])),
			Position: Vector{float64(X), float64(Y)},
		}
		gs.Projectiles = append(gs.Projectiles, &p)
	}
	return gs
}

// TODO: Map objects should be dynamic and passed from server to client on init

type MapObject struct {
	BottmLeft Vector
	TopRight  Vector
	IsRigid   bool
	IsVisible bool
}

func (mo *MapObject) IsWithinX(v Vector) bool {
	return v.X >= mo.BottmLeft.X && v.X < mo.TopRight.X
}

func (mo *MapObject) IsWithinY(v Vector) bool {
	return v.Y >= mo.BottmLeft.Y && v.Y < mo.TopRight.Y
}

func (mo *MapObject) CollidesWith(v Vector) bool {
	return mo.IsRigid &&
		v.X >= mo.BottmLeft.X &&
		v.X < mo.TopRight.X &&
		v.Y >= mo.BottmLeft.Y &&
		v.Y < mo.TopRight.Y
}

var MapObjects = []MapObject{
	{BottmLeft: Vector{X: 10, Y: 0}, TopRight: Vector{X: 15, Y: 10}, IsRigid: true, IsVisible: true},
	{BottmLeft: Vector{X: 17, Y: 15}, TopRight: Vector{X: 30, Y: 18}, IsRigid: true, IsVisible: true},

	// Map borders
	{BottmLeft: Vector{X: -1, Y: -1}, TopRight: Vector{X: FieldMaxX + 1, Y: 0}, IsRigid: true},                    // Bottom
	{BottmLeft: Vector{X: -1, Y: FieldMaxY}, TopRight: Vector{X: FieldMaxX + 1, Y: FieldMaxY + 1}, IsRigid: true}, // Top
	{BottmLeft: Vector{X: -1, Y: -1}, TopRight: Vector{X: 0, Y: FieldMaxY + 1}, IsRigid: true},                    // Left
	{BottmLeft: Vector{X: FieldMaxX, Y: -1}, TopRight: Vector{X: FieldMaxX + 1, Y: FieldMaxY + 1}, IsRigid: true}, // Left
}
