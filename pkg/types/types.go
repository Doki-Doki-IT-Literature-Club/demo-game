package types

type Command = int

const (
	UP = iota
	DOWN
	LEFT
	RIGHT
)

type Player struct {
	PlayerRune rune
	X          int
	Y          int
}

type GameState struct {
	Players []Player
}
