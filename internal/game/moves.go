package game

// Steps a piece takes from a square: row then column.
var (
	straightSteps = [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
	diagonalSteps = [][2]int{{-1, -1}, {-1, 1}, {1, -1}, {1, 1}}
	royalSteps    = append(append([][2]int{}, straightSteps...), diagonalSteps...)
	knightSteps   = [][2]int{{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2}, {1, -2}, {1, 2}, {2, -1}, {2, 1}}
)

// Moves are the legal moves for the piece on a square: everything it could do,
// less anything that would leave its own king attacked.
func (b *Board) Moves(row, col int) []Move {
	piece := b.Piece(row, col)
	if piece == 0 {
		return nil
	}

	side := sideOf(piece)
	candidates := pseudoMoves(&b.pieces, row, col, b.passantRow, b.passantCol)

	if kind(piece) == 'K' {
		candidates = append(candidates, b.castleMoves(row, col, side)...)
	}

	moves := make([]Move, 0, len(candidates))

	for _, m := range candidates {
		if !b.exposesKing(m, side) {
			moves = append(moves, m)
		}
	}

	return moves
}

// exposesKing plays a move on a copy of the board and reports whether side's
// king ends up attacked. The pieces are an array, so the copy is the whole
// undo.
func (b *Board) exposesKing(m Move, side Side) bool {
	pieces := b.pieces
	applyTo(&pieces, m, b.passantRow, b.passantCol)

	row, col := kingSquare(&pieces, side)

	return onBoard(row, col) && attacked(&pieces, row, col, side.Other())
}

// inCheck reports whether side's king is under attack.
func (b *Board) inCheck(side Side) bool {
	row, col := kingSquare(&b.pieces, side)

	return onBoard(row, col) && attacked(&b.pieces, row, col, side.Other())
}

// hasMoves reports whether side has anything legal left to play.
func (b *Board) hasMoves(side Side) bool {
	for row := range size {
		for col := range size {
			if b.holds(row, col, side) && len(b.Moves(row, col)) > 0 {
				return true
			}
		}
	}

	return false
}

// castleMoves are the castles the king on a square can still make. It needs to
// know what has moved, so unlike the rest it works off the board, not a bare
// position.
func (b *Board) castleMoves(row, col int, side Side) []Move {
	if b.moved[row][col] || b.inCheck(side) {
		return nil
	}

	var moves []Move

	for _, side1 := range [2]struct{ rook, step int }{{0, -1}, {size - 1, 1}} {
		if b.moved[row][side1.rook] || b.pieces[row][side1.rook] != pieceOf(side, 'R') {
			continue
		}

		if !clearBetween(&b.pieces, row, col, side1.rook) {
			continue
		}

		// The king may not cross a square that is attacked. Where it lands is
		// left to the usual check on every move.
		if attacked(&b.pieces, row, col+side1.step, side.Other()) {
			continue
		}

		moves = append(moves, move(row, col, row, col+2*side1.step))
	}

	return moves
}

// clearBetween reports whether the squares strictly between two columns of a
// rank are empty.
func clearBetween(pieces *[size][size]byte, row, from, to int) bool {
	for col := min(from, to) + 1; col < max(from, to); col++ {
		if pieces[row][col] != 0 {
			return false
		}
	}

	return true
}

// pseudoMoves are the moves a piece could make if its own king's safety didn't
// matter. Castling isn't among them: it turns on what has moved, which a bare
// position doesn't record.
func pseudoMoves(pieces *[size][size]byte, row, col, passantRow, passantCol int) []Move {
	piece := at(pieces, row, col)
	if piece == 0 {
		return nil
	}

	side := sideOf(piece)

	switch kind(piece) {
	case 'P':
		return pawnMoves(pieces, row, col, side, passantRow, passantCol)
	case 'N':
		return stepMoves(pieces, row, col, side, knightSteps)
	case 'K':
		return stepMoves(pieces, row, col, side, royalSteps)
	case 'B':
		return slideMoves(pieces, row, col, side, diagonalSteps)
	case 'R':
		return slideMoves(pieces, row, col, side, straightSteps)
	case 'Q':
		return slideMoves(pieces, row, col, side, royalSteps)
	}

	return nil
}

// slideMoves runs along each step until the edge or a piece. The first enemy in
// the way can be taken, anything else stops the run.
func slideMoves(pieces *[size][size]byte, row, col int, side Side, steps [][2]int) []Move {
	var moves []Move

	for _, step := range steps {
		r, c := row+step[0], col+step[1]

		for onBoard(r, c) {
			if piece := pieces[r][c]; piece != 0 {
				if sideOf(piece) != side {
					moves = append(moves, move(row, col, r, c))
				}

				break
			}

			moves = append(moves, move(row, col, r, c))
			r, c = r+step[0], c+step[1]
		}
	}

	return moves
}

// stepMoves is one square along each step, for knights and kings.
func stepMoves(pieces *[size][size]byte, row, col int, side Side, steps [][2]int) []Move {
	var moves []Move

	for _, step := range steps {
		r, c := row+step[0], col+step[1]
		if !onBoard(r, c) {
			continue
		}

		if piece := pieces[r][c]; piece != 0 && sideOf(piece) == side {
			continue
		}

		moves = append(moves, move(row, col, r, c))
	}

	return moves
}

// pawnMoves walks forward onto empty squares and takes diagonally, including
// onto the square a pawn just stepped over.
func pawnMoves(pieces *[size][size]byte, row, col int, side Side, passantRow, passantCol int) []Move {
	var moves []Move

	forward, home := -1, size-2
	if side == Black {
		forward, home = 1, 1
	}

	if vacant(pieces, row+forward, col) {
		moves = append(moves, move(row, col, row+forward, col))

		if row == home && vacant(pieces, row+2*forward, col) {
			moves = append(moves, move(row, col, row+2*forward, col))
		}
	}

	for _, step := range [2]int{-1, 1} {
		r, c := row+forward, col+step
		if !onBoard(r, c) {
			continue
		}

		taken := pieces[r][c]
		if (taken != 0 && sideOf(taken) != side) || (r == passantRow && c == passantCol) {
			moves = append(moves, move(row, col, r, c))
		}
	}

	return moves
}

// applyTo plays a move on a position, including the parts that shift a second
// piece: the rook in a castle and the pawn taken in passing. It reports the new
// en passant square, which is off the board unless a pawn just stepped two.
func applyTo(pieces *[size][size]byte, m Move, passantRow, passantCol int) (int, int) {
	piece := pieces[m.FromRow][m.FromCol]

	pieces[m.FromRow][m.FromCol] = 0
	pieces[m.ToRow][m.ToCol] = piece

	switch kind(piece) {
	case 'P':
		// Taken in passing: the pawn stands beside the square moved to.
		if m.ToRow == passantRow && m.ToCol == passantCol && m.FromCol != m.ToCol {
			pieces[m.FromRow][m.ToCol] = 0
		}

		if promotes(piece, m.ToRow) {
			pieces[m.ToRow][m.ToCol] = pieceOf(sideOf(piece), promotionOf(m))
		}

		if abs(m.ToRow-m.FromRow) == 2 {
			return (m.FromRow + m.ToRow) / 2, m.FromCol
		}

	case 'K':
		// Castling: the king steps two and the rook hops to its other side.
		if abs(m.ToCol-m.FromCol) == 2 {
			from, to := 0, 3
			if m.ToCol > m.FromCol {
				from, to = size-1, 5
			}

			pieces[m.FromRow][to] = pieces[m.FromRow][from]
			pieces[m.FromRow][from] = 0
		}
	}

	return -1, -1
}

// attacked reports whether side has a piece bearing on a square.
func attacked(pieces *[size][size]byte, row, col int, side Side) bool {
	// A pawn of side attacking this square stands one rank behind it.
	back := 1
	if side == Black {
		back = -1
	}

	for _, step := range [2]int{-1, 1} {
		if at(pieces, row+back, col+step) == pieceOf(side, 'P') {
			return true
		}
	}

	for _, step := range knightSteps {
		if at(pieces, row+step[0], col+step[1]) == pieceOf(side, 'N') {
			return true
		}
	}

	for _, step := range royalSteps {
		if at(pieces, row+step[0], col+step[1]) == pieceOf(side, 'K') {
			return true
		}
	}

	return slidesTo(pieces, row, col, side, straightSteps, 'R') ||
		slidesTo(pieces, row, col, side, diagonalSteps, 'B')
}

// slidesTo looks along each step for the nearest piece, and reports whether it
// is one of side's queens or the long-range piece that runs that way.
func slidesTo(pieces *[size][size]byte, row, col int, side Side, steps [][2]int, long byte) bool {
	for _, step := range steps {
		r, c := row+step[0], col+step[1]

		for onBoard(r, c) {
			if piece := pieces[r][c]; piece != 0 {
				if piece == pieceOf(side, long) || piece == pieceOf(side, 'Q') {
					return true
				}

				break
			}

			r, c = r+step[0], c+step[1]
		}
	}

	return false
}

// kingSquare is where side's king stands, or off the board if it has none.
func kingSquare(pieces *[size][size]byte, side Side) (int, int) {
	king := pieceOf(side, 'K')

	for row := range size {
		for col := range size {
			if pieces[row][col] == king {
				return row, col
			}
		}
	}

	return -1, -1
}

// canReach reports whether one of moves ends on a square.
func canReach(moves []Move, row, col int) bool {
	for _, m := range moves {
		if m.ToRow == row && m.ToCol == col {
			return true
		}
	}

	return false
}

// holds reports whether a square carries one of side's pieces.
func (b *Board) holds(row, col int, side Side) bool {
	piece := b.Piece(row, col)
	return piece != 0 && sideOf(piece) == side
}

func vacant(pieces *[size][size]byte, row, col int) bool {
	return onBoard(row, col) && pieces[row][col] == 0
}

func at(pieces *[size][size]byte, row, col int) byte {
	if !onBoard(row, col) {
		return 0
	}

	return pieces[row][col]
}

func onBoard(row, col int) bool {
	return row >= 0 && row < size && col >= 0 && col < size
}

// kind is a piece without its side, so 'p' and 'P' are both pawns.
func kind(piece byte) byte {
	if piece >= 'a' && piece <= 'z' {
		return piece - 'a' + 'A'
	}

	return piece
}

// pieceOf is side's version of a piece kind, which is given uppercase.
func pieceOf(side Side, k byte) byte {
	if side == Black {
		return k - 'A' + 'a'
	}

	return k
}

// promotes reports whether a move lands a pawn on the far rank.
func promotes(piece byte, toRow int) bool {
	return kind(piece) == 'P' && (toRow == 0 || toRow == size-1)
}

// promotionOf is what a move promotes to, a queen unless it says otherwise.
func promotionOf(m Move) byte {
	if m.Promote == 0 {
		return 'Q'
	}

	return m.Promote
}

// insufficient reports whether neither side is left with the material to mate.
// Two knights are not counted out: they cannot force mate, but mate is still
// possible, so the game is not dead.
func insufficient(pieces *[size][size]byte) bool {
	var whiteMinors, blackMinors int
	whiteBishop, blackBishop := -1, -1

	for row := range size {
		for col := range size {
			piece := pieces[row][col]
			if piece == 0 || kind(piece) == 'K' {
				continue
			}

			white := sideOf(piece) == White

			switch kind(piece) {
			case 'B':
				if white {
					whiteMinors, whiteBishop = whiteMinors+1, (row+col)%2
				} else {
					blackMinors, blackBishop = blackMinors+1, (row+col)%2
				}

			case 'N':
				if white {
					whiteMinors++
				} else {
					blackMinors++
				}

			default:
				// A pawn, rook or queen can still force mate.
				return false
			}
		}
	}

	if whiteMinors > 1 || blackMinors > 1 {
		return false
	}

	// One piece each is only dead when both are bishops on the same colour.
	if whiteMinors == 1 && blackMinors == 1 {
		return whiteBishop >= 0 && whiteBishop == blackBishop
	}

	return true
}

func move(fromRow, fromCol, toRow, toCol int) Move {
	return Move{FromRow: fromRow, FromCol: fromCol, ToRow: toRow, ToCol: toCol}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}

	return n
}
