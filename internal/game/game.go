package game

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

type Game struct {
	board *Board
	white Player
	black Player
}

func New() *Game {
	return NewGame(
		NewBoard(startingPosition),
		NewHuman(White),
		NewHuman(Black),
	)
}

func NewGame(board *Board, white, black Player) *Game {
	g := &Game{board: board, white: white, black: black}

	board.onMove = g.playEngines
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
	New().board.Display()
}
