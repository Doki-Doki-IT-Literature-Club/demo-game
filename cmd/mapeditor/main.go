package main

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"

	types "github.com/Doki-Doki-IT-Literature-Club/demo-game/pkg/types"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	fieldMaxX uint32
	fieldMaxY uint32

	cursorPos types.Vector

	cursorPosRune   rune
	emptyFiledRune  rune
	wallPreviewRune rune
	wallRune        rune

	wallInitPoint *types.Vector
	walls         []types.MapObject
}

func InitModel() model {
	return model{
		fieldMaxX:       500,
		fieldMaxY:       500,
		cursorPos:       types.Vector{X: 0, Y: 0},
		emptyFiledRune:  ' ',
		cursorPosRune:   '@',
		wallPreviewRune: '.',
		wallRune:        'w',
		walls:           []types.MapObject{},
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
		case "up", "k":
			m.cursorPos.Y++
			return m, nil
		case "down", "j":
			m.cursorPos.Y--
			return m, nil
		case "left", "h":
			m.cursorPos.X--
			return m, nil
		case "right", "l":
			m.cursorPos.X++
			return m, nil
		case "w":
			if m.wallInitPoint == nil {
				m.wallInitPoint = &m.cursorPos
			} else {
				miny := min(m.wallInitPoint.Y, m.cursorPos.Y)
				maxy := max(m.wallInitPoint.Y, m.cursorPos.Y) + 1
				minx := min(m.wallInitPoint.X, m.cursorPos.X)
				maxx := max(m.wallInitPoint.X, m.cursorPos.X) + 1
				m.walls = append(m.walls, types.MapObject{
					Position:      types.Vector{X: minx, Y: miny},
					CollisionArea: types.CollisionArea{X: maxx - minx, Y: maxy - miny},
					IsVisible:     true,
				})
				m.wallInitPoint = nil
			}
			return m, nil
		case "s":
			b, err := json.Marshal(m.walls)
			if err != nil {
				panic(err)
			}
			f, err := os.Create("map.json")
			if err != nil {
				panic(err)
			}
			_, err = f.Write(b)
			if err != nil {
				panic(err)
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	// Compose empty filed
	field := [][]rune{}
	for range m.fieldMaxY {
		row := []rune{}
		for range m.fieldMaxX {
			row = append(row, m.emptyFiledRune)
		}
		field = append(field, row)
	}

	// Draw wall preview
	if m.wallInitPoint != nil {
		miny := int(min(m.wallInitPoint.Y, m.cursorPos.Y))
		maxy := int(max(m.wallInitPoint.Y, m.cursorPos.Y))
		minx := int(min(m.wallInitPoint.X, m.cursorPos.X))
		maxx := int(max(m.wallInitPoint.X, m.cursorPos.X))
		for dy := range maxy - miny + 1 {
			y := dy + miny
			for dx := range maxx - minx + 1 {
				x := dx + minx
				field[y][x] = m.wallPreviewRune
			}
		}
	}

	// Draw walls
	for _, wall := range m.walls {
		for dy := range int(wall.CollisionArea.Y) {
			y := dy + int(wall.Position.Y)
			for dx := range int(wall.CollisionArea.X) {
				x := dx + int(wall.Position.X)
				field[y][x] = m.wallRune
			}
		}
	}

	// Draw cursor
	field[int(m.cursorPos.Y)][int(m.cursorPos.X)] = m.cursorPosRune

	res := ""
	slices.Reverse(field)
	for _, row := range field {
		res += string(row) + "\n"
	}
	return res
}

func main() {
	m := InitModel()

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
