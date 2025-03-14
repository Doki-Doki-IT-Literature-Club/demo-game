package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	game Game
}

func initialModel() model {
	return model{
		Game{
			field_x:        50,
			field_y:        30,
			emptyFiledRune: '.',
			players: []Player{
				Player{x: 3, y: 5, playerRune: 'K'},
				Player{x: 14, y: 27, playerRune: 'S'},
			},
		},
	}
}
func (m model) Init() tea.Cmd {
	return nil
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:

		switch msg.String() {

		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() string {
	return m.game.Render()
}

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
}

type Comand struct {
  asd string,
}

func (g *Game) ProcessCommand (cmd Comand) {
  switch cmd.type {
  case "d":
    
  }

}

func debug(s string) {
	fmt.Printf("--->%s<---", s)
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

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
