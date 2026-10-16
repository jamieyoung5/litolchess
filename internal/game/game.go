package game

import (
	"fmt"
	"os"
	"time"

	"github.com/jamieyoung5/stgui"
	"github.com/jamieyoung5/stgui/events"
)

var startingPosition = [size]string{
	"rnbqkbnr",
	"pppppppp",
	"........",
	"........",
	"........",
	"........",
	"PPPPPPPP",
	"RNBQKBNR",
}

// tick is a second going by. The clocks run off it.
type tick struct {
	events.Msg
}

type Game struct {
	board *Board
	white Player
	black Player
}

func New(limit time.Duration) *Game {
	return NewGame(
		NewBoard(startingPosition, NewClock(limit)),
		NewHuman(White),
		NewHuman(Black),
	)
}

func NewGame(board *Board, white, black Player) *Game {
	g := &Game{board: board, white: white, black: black}

	board.onMove = g.playEngines
	board.automatic = func(side Side) bool {
		_, ok := g.player(side).(Engine)
		return ok
	}

	g.playEngines()

	return g
}

func (g *Game) player(side Side) Player {
	if side == White {
		return g.white
	}
	return g.black
}

// playEngines lets engines answer until it's a human's turn again. Two engines
// would play the whole game here, without the screen catching up in between.
func (g *Game) playEngines() {
	for {
		engine, ok := g.player(g.board.Turn()).(Engine)
		if !ok {
			return
		}

		g.board.Apply(engine.Move(g.board))
	}
}

func Play() {
	// The game being played, if any. Written when the menu starts one and read
	// when a tick arrives, both on the event loop.
	var current *Game

	screen := menu(func(limit time.Duration) stgui.Action {
		current = New(limit)
		return stgui.Push(current.board.Screen())
	})

	app := stgui.NewApp(screen).QuitOn("q").WithMouse().OnEvent(func(event events.Event) stgui.Action {
		if _, ok := event.(tick); ok && current != nil {
			current.board.Tick(time.Second)
		}

		return stgui.Stay()
	})

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			app.Send(tick{})
		}
	}()

	if err := app.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
