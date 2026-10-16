package game

// Side is the army a player commands.
type Side byte

const (
	White Side = 'w'
	Black Side = 'b'
)

func (s Side) String() string {
	if s == White {
		return "white"
	}
	return "black"
}

// Other is the side that moves next.
func (s Side) Other() Side {
	if s == White {
		return Black
	}
	return White
}

// Move is a piece going from one square to another.
type Move struct {
	FromRow, FromCol int
	ToRow, ToCol     int

	// Promote is what a pawn reaching the far rank becomes, given uppercase.
	// Zero means a queen, so a move needs it only for an underpromotion.
	Promote byte
}

// Player commands one side.
type Player interface {
	Side() Side
}

// Engine is a player that picks its own moves. A player that isn't one moves
// with the board's cursor instead.
type Engine interface {
	Player
	Move(board *Board) Move
}

// Human moves with the cursor, so it has no Move of its own.
type Human struct {
	side Side
}

func NewHuman(side Side) *Human {
	return &Human{side: side}
}

func (h *Human) Side() Side {
	return h.side
}

// sideOf is which army a piece belongs to. Uppercase is white.
func sideOf(piece byte) Side {
	if piece >= 'a' && piece <= 'z' {
		return Black
	}
	return White
}
