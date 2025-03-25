package types

import (
	"encoding/binary"
)

type PlayerID uint32

type Command byte

const (
	UP    = 0x01
	DOWN  = 0x02
	LEFT  = 0x03
	RIGHT = 0x04
)

const FieldMaxX = 50
const FieldMaxY = 30

type Vector struct {
	X int32
	Y int32
}

func (v *Vector) Add(other Vector) {
	v.X += other.X
	v.Y += other.Y
}

type Player struct {
	ID         PlayerID
	PlayerRune rune
	X          uint32
	Y          uint32
	Vec        Vector
}

func (p *Player) ApplyVec() {
	tx := int32(p.X) + p.Vec.X
	ty := int32(p.Y) + p.Vec.Y

	p.X = uint32(min(max(0, tx), FieldMaxX-1))
	p.Y = uint32(min(max(0, ty), FieldMaxY-1))
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
		binary.BigEndian.PutUint32(pb[8:12], uint32(p.X))
		binary.BigEndian.PutUint32(pb[12:], uint32(p.Y))
		res = append(res, pb[:]...)
	}
	return res
}

func GameStateFromBytes(data []byte) GameState {
	gs := GameState{map[PlayerID]*Player{}}
	for i := 0; i < len(data)/16; i++ {
		k := i * 16

		playerID := PlayerID(binary.BigEndian.Uint32(data[k : k+4]))
		p := Player{
			ID:         playerID,
			PlayerRune: rune(binary.BigEndian.Uint32(data[k+4 : k+8])),
			X:          binary.BigEndian.Uint32(data[k+8 : k+12]),
			Y:          binary.BigEndian.Uint32(data[k+12 : k+16]),
		}
		gs.Players[playerID] = &p
	}
	return gs
}
