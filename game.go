package main

import (
	"fmt"
	"time"
)

type Command = int

const (
	UP = iota
	DOWN
	LEFT
	RIGHT
)

type Player struct {
	playerRune rune
	x          int
	y          int
}

type Game struct {
	field_x        int
	field_y        int
	emptyFiledRune rune
	players        []Player
	playersChan    <-chan []Player
	commandsChan   chan<- Command
}

func (g *Game) Render() string {
	field := [][]rune{}
	for range g.field_y {
		row := []rune{}
		for range g.field_x {
			row = append(row, g.emptyFiledRune)
		}
		field = append(field, row)
	}

	for _, p := range g.players {
		field[p.y][p.x] = p.playerRune
	}

	res := ""
	for _, row := range field {
		res += string(row) + "\n"
	}
	return res
}

func (g *Game) MoveMe(direction Command) {
	g.commandsChan <- direction
}

// "Mocked" runMockServer from which we receive game state,
// and to which we send commands
func runMockServer() (<-chan []Player, chan<- Command) {
	pch := make(chan []Player)
	cch := make(chan Command)

	players := []Player{
		{x: 3, y: 5, playerRune: 'K'},
		{x: 14, y: 27, playerRune: 'S'},
	}

	go func() {
		mockBotMovementTicker := time.NewTicker(time.Second)
		for {
			select {
			case <-mockBotMovementTicker.C:
				players[1].y--
				pch <- players
			case direction := <-cch:
				switch direction {
				case UP:
					players[0].y--
				case DOWN:
					players[0].y++
				case LEFT:
					players[0].x--
				case RIGHT:
					players[0].x++
				}
				pch <- players
			}
		}
	}()

	return pch, cch
}

func debug(s string) {
	fmt.Printf("--->%s<---", s)
}
