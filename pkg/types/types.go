package types

import (
	"encoding/binary"
	"math"
)

type PlayerID uint32

type Command byte

const (
	UP    = 0x01
	DOWN  = 0x02
	LEFT  = 0x03
	RIGHT = 0x04
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

type Player struct {
	ID         PlayerID
	PlayerRune rune
	Position   Vector
	Speed      Vector
}

type GameState struct {
	Players map[PlayerID]*Player
}

func (gs *GameState) ToBytes() []byte {
	res := []byte{byte(len(gs.Players))}
	for _, p := range gs.Players {
		pb := [16]byte{}
		binary.BigEndian.PutUint32(pb[:4], uint32(p.ID))
		binary.BigEndian.PutUint32(pb[4:8], uint32(p.PlayerRune))
		binary.BigEndian.PutUint32(pb[8:12], uint32(p.Position.X))
		binary.BigEndian.PutUint32(pb[12:], uint32(p.Position.Y))
		res = append(res, pb[:]...)
	}
	return res
}

func GameStateFromBytes(data []byte) GameState {
	gs := GameState{map[PlayerID]*Player{}}
	for i := 0; i < len(data)/16; i++ {
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
	return gs
}
