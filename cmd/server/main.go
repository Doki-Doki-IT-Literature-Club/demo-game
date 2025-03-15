package main

import (
	"fmt"
	"net"
	"sync"
)

type ClinetConn struct {
	// TODO: state struct
	write chan<- []byte
}

type Player struct {
	x int
	y int
}

type GameState struct {
	players []Player
}

type GameEngine struct {
	conns []*ClinetConn
	state GameState
	// TODO: commands struct
	engineInput chan []byte

	mu sync.Mutex
}

func (ge *GameEngine) addPlayer(conn *ClinetConn) {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	ge.conns = append(ge.conns, conn)
	ge.state.players = append(
		ge.state.players,
		Player{
			x: len(ge.state.players),
			y: len(ge.state.players),
		},
	)
}

func (ge *GameEngine) HangleConnection(conn net.Conn) {
	fmt.Printf("New connection: %v\n", conn)

	write := make(chan []byte)

	cliConn := &ClinetConn{write}

	ge.addPlayer(cliConn)

	go func() {
		for data := range write {
			// TODO struct -> bytes
			_, err := conn.Write(data)
			if err != nil {
				panic(err)
			}
		}
	}()

	go func() {
		for {
			buff := [1024]byte{}
			n, err := conn.Read(buff[:])
			if err != nil {
				panic(err)
			}
			// TODO make structs from bytes
			ge.engineInput <- buff[:n]
		}
	}()
	// TODO: handle connection close
}

func (ge *GameEngine) Run() {
	for data := range ge.engineInput {
		// TODO: handle commands
		fmt.Printf("Got data in Run: %v\n", data)
		for _, cli := range ge.conns {
			cli.write <- data
		}
	}
}

func RunGameEngine() *GameEngine {
	ge := &GameEngine{
		state:       GameState{players: []Player{}},
		conns:       []*ClinetConn{},
		engineInput: make(chan []byte),
	}
	go ge.Run()
	return ge
}

func main() {

	fmt.Println("starting server")

	ge := RunGameEngine()

	listner, err := net.Listen("tcp", "0.0.0.0:8000")
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
