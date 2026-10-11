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
}

func New() *Game {
	return NewGame(
		NewBoard(startingPosition),
	)
}

func NewGame(board *Board) *Game {
	return &Game{
		board: board,
	}
}

func Play() {
	New().board.Display()
}
