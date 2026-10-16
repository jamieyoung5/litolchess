package game

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jamieyoung5/stgui"
	"github.com/jamieyoung5/stgui/widgets"
)

const (
	size = 8

	// The cursor starts on e2.
	startRow, startCol = size - 2, 4

	reset = "\033[0m"

	// A middle dot is Latin-1, so it draws anywhere the pieces do.
	dot = "\u00b7"

	// fiftyMoves is the rule in halfmoves: fifty each.
	fiftyMoves = 100
)

// squareSizes are the shapes a square can be drawn at, smallest first: columns
// then rows. Three columns is the floor, since the selector needs a bracket
// either side of the piece. Widths are odd so the piece centres exactly, and
// roughly twice the height because a terminal cell is about twice as tall as
// it is wide.
var squareSizes = [][2]int{{3, 1}, {5, 2}, {7, 3}, {9, 4}}

// castleSquares are the king and rook homes, the only squares whose history
// changes what a position allows.
var castleSquares = [][2]int{{7, 4}, {7, 0}, {7, 7}, {0, 4}, {0, 0}, {0, 7}}

// Dark squares invert whatever the terminal already draws with, so the board
// suits any colour scheme and needs no palette of its own.
const (
	styleDarkSquare = stgui.StyleReverse
	styleCoord      = stgui.StyleDim

	// The selector is drawn as brackets, so this only has to thicken them.
	// It must be non-empty or the grid falls back to its own highlight.
	styleCursor = stgui.StyleBold
)

// Outlined pieces are white and filled pieces are black
var glyphs = map[byte]rune{
	'K': '♔', 'Q': '♕', 'R': '♖', 'B': '♗', 'N': '♘', 'P': '♙',
	'k': '♚', 'q': '♛', 'r': '♜', 'b': '♝', 'n': '♞', 'p': '♟',
}

type Board struct {
	grid   *stgui.Grid
	pieces [size][size]byte
	status *widgets.Label

	turn  Side
	clock *Clock

	// over is set once the game is decided. winner only means anything then,
	// and not when drawn is set.
	over   bool
	drawn  bool
	winner Side

	// passantRow and passantCol are the square a pawn just stepped over, off
	// the board when there isn't one.
	passantRow, passantCol int

	// moved records squares whose piece has been disturbed, which is all
	// castling needs to know.
	moved [size][size]bool

	// halfmoves since the last pawn move or capture, for the fifty-move rule.
	halfmoves int

	// seen counts how often each position has come up, and repeats is the
	// count for the one on the board now.
	seen    map[string]int
	repeats int

	// offer is the side with a draw on the table, zero when there isn't one.
	offer Side

	// history is the positions a move can be taken back to.
	history []snapshot

	// automatic reports whether a side plays itself, so undo knows to step
	// back past its reply. Nil means both sides are played by hand.
	automatic func(Side) bool

	// action is the last thing that happened, kept so a clock tick can redraw
	// the status without losing it.
	action string

	// holding is zero when nothing is picked up; legal is where it can go.
	holding          byte
	fromRow, fromCol int
	legal            []Move

	curRow, curCol int

	// size indexes squareSizes.
	size int

	// onMove runs after a completed move, so the game can let an engine answer.
	onMove func()
}

func NewBoard(startingPos [size]string, clock *Clock) *Board {
	w := squareSizes[0][0]

	style := &stgui.GridStyle{
		NoValue:       strings.Repeat(" ", w),
		MinCellWidth:  w,
		SelectedStyle: styleCursor,
	}

	b := &Board{
		grid:       stgui.NewEmptyGrid(size, size, style),
		status:     widgets.NewLabel(""),
		turn:       White,
		clock:      clock,
		passantRow: -1,
		passantCol: -1,
		seen:       map[string]int{},
		curRow:     startRow,
		curCol:     startCol,
	}

	for row, rank := range startingPos {
		for col := range rank {
			if rank[col] != '.' {
				b.pieces[row][col] = rank[col]
			}
		}
	}

	b.seen[b.key()] = 1
	b.repeats = 1

	b.setStatus("enter: pick up   u: undo   d: draw\nr: resign   +/-: size   q: quit")
	b.draw()

	return b
}

// Screen is the board ready to be shown.
func (b *Board) Screen() *stgui.Screen {
	container := widgets.NewContainer(b.grid).OnKey(b.onKey)
	board := view{Container: container, board: b}

	return stgui.NewScreen(stgui.NewCursor(board, b.grid, b.curRow, b.curCol),
		stgui.Row(board),
		stgui.Row(b.status),
	)
}

func (b *Board) draw() {
	b.grid.Each(func(row, col int, cell *stgui.Cell) {
		piece := b.pieces[row][col]

		// A piece in hand is off the board until it's put down, which marks
		// where it came from without a second style.
		if b.holding != 0 && row == b.fromRow && col == b.fromCol {
			piece = 0
		}

		w, h := b.square()

		b.grid.Set(row, col, squareText(piece,
			canReach(b.legal, row, col),
			row == b.curRow && col == b.curCol, w, h))

		cell.Style = ""
		if (row+col)%2 == 1 {
			cell.Style = styleDarkSquare
		}
	})
}

func (b *Board) onKey(cursor *stgui.Cursor, input stgui.Input) stgui.Action {
	if key, ok := input.Rune(); ok {
		switch key {
		case 'u':
			b.Undo()
		case 'd':
			b.proposeDraw()
		case 'r':
			b.resign()
		case '+', '=':
			b.Resize(1)
		case '-', '_':
			b.Resize(-1)
		}

		return stgui.Stay()
	}

	if !input.Is(stgui.ControlSelect) {
		return stgui.Stay()
	}

	// Nothing left to play, so enter goes back to the menu.
	if b.over {
		return stgui.Close()
	}

	row, col := cursor.Row, cursor.Col

	switch {
	case b.holding == 0:
		piece := b.pieces[row][col]

		switch {
		case piece == 0:
			b.setStatus(fmt.Sprintf("%s is empty", square(row, col)))
		case sideOf(piece) != b.turn:
			b.setStatus(fmt.Sprintf("%s is %s's", square(row, col), sideOf(piece)))
		default:
			legal := b.Moves(row, col)
			if len(legal) == 0 {
				b.setStatus(fmt.Sprintf("%s on %s has nowhere to go", glyph(piece), square(row, col)))
				break
			}

			b.holding = piece
			b.fromRow, b.fromCol = row, col
			b.legal = legal
			b.setStatus(fmt.Sprintf("holding %s from %s", glyph(piece), square(row, col)))
		}

	case row == b.fromRow && col == b.fromCol:
		b.setStatus(fmt.Sprintf("put %s back on %s", glyph(b.holding), square(row, col)))
		b.holding = 0
		b.legal = nil

	default:
		if !canReach(b.legal, row, col) {
			b.setStatus(fmt.Sprintf("%s can't go to %s", glyph(b.holding), square(row, col)))
			break
		}

		m := move(b.fromRow, b.fromCol, row, col)

		// A pawn reaching the far rank gets a say in what it becomes.
		if promotes(b.holding, row) {
			return stgui.Push(promotionMenu(sideOf(b.holding), func(piece byte) stgui.Action {
				m.Promote = piece
				b.play(m)

				return stgui.Close()
			}))
		}

		b.play(m)
	}

	return stgui.Stay()
}

// play makes a move and lets the game answer it.
func (b *Board) play(m Move) {
	b.holding = 0
	b.legal = nil

	b.Apply(m)

	if b.onMove != nil {
		b.onMove()
	}
}

// Turn is the side to move.
func (b *Board) Turn() Side {
	return b.turn
}

// Piece is what stands on a square, or zero if nothing does. Squares off the
// board are empty, so an engine can look at a neighbour without checking first.
func (b *Board) Piece(row, col int) byte {
	if row < 0 || row >= size || col < 0 || col >= size {
		return 0
	}

	return b.pieces[row][col]
}

// Apply moves a piece and passes the turn. It asks no questions: whether the
// move is legal is not the board's business.
func (b *Board) Apply(m Move) {
	piece := b.pieces[m.FromRow][m.FromCol]
	taken := b.pieces[m.ToRow][m.ToCol]

	// Taken in passing, so the square moved to was empty and the pawn that
	// goes is the one alongside.
	if kind(piece) == 'P' && m.ToRow == b.passantRow && m.ToCol == b.passantCol && m.FromCol != m.ToCol {
		taken = b.pieces[m.FromRow][m.ToCol]
	}

	castled := kind(piece) == 'K' && abs(m.ToCol-m.FromCol) == 2

	var promoted byte
	if promotes(piece, m.ToRow) {
		promoted = pieceOf(sideOf(piece), promotionOf(m))
	}

	b.save()

	b.moved[m.FromRow][m.FromCol] = true
	b.moved[m.ToRow][m.ToCol] = true

	b.passantRow, b.passantCol = applyTo(&b.pieces, m, b.passantRow, b.passantCol)

	b.turn = b.turn.Other()

	// A pawn move or a capture can never be undone, so both reset the count
	// and make every earlier position unreachable.
	if kind(piece) == 'P' || taken != 0 {
		b.halfmoves = 0
		clear(b.seen)
	} else {
		b.halfmoves++
	}

	key := b.key()
	b.seen[key]++
	b.repeats = b.seen[key]

	// Moving is how the side that was asked turns a draw down.
	if b.offer == b.turn {
		b.offer = 0
	}

	text := moveText(piece, taken, m, castled, promoted)
	b.describe(text)
	b.setStatus(text)
	b.finish()
	b.draw()
}

// moveText is what just happened, as a line for the status.
func moveText(piece, taken byte, m Move, castled bool, promoted byte) string {
	if castled {
		side := "kingside"
		if m.ToCol < m.FromCol {
			side = "queenside"
		}

		return fmt.Sprintf("%s castles %s", glyph(piece), side)
	}

	to := square(m.ToRow, m.ToCol)

	switch {
	case taken != 0 && promoted != 0:
		return fmt.Sprintf("%s takes %s on %s, becomes %s", glyph(piece), glyph(taken), to, glyph(promoted))
	case promoted != 0:
		return fmt.Sprintf("%s to %s, becomes %s", glyph(piece), to, glyph(promoted))
	case taken != 0:
		return fmt.Sprintf("%s takes %s on %s", glyph(piece), glyph(taken), to)
	}

	return fmt.Sprintf("%s to %s", glyph(piece), to)
}

// finish ends the game when the side to move has nothing legal left: mate if
// its king is attacked, stalemate if not.
func (b *Board) finish() {
	switch {
	case !b.hasMoves(b.turn):
		if b.inCheck(b.turn) {
			b.winner = b.turn.Other()
			b.conclude("checkmate")

			return
		}

		b.conclude("stalemate")

	case insufficient(&b.pieces):
		b.conclude("dead position")

	case b.repeats >= 3:
		b.conclude("threefold repetition")

	case b.halfmoves >= fiftyMoves:
		b.conclude("fifty-move rule")
	}
}

// conclude ends the game, keeping the move that did it. Anything but a win by
// the side that just moved is a draw.
func (b *Board) conclude(outcome string) {
	b.over = true
	b.drawn = b.winner == 0

	// Keep the move that ended it, rather than replacing it.
	if b.action != "" {
		outcome = b.action + ", " + outcome
	}

	b.setStatus(outcome)
}

// end stops the game with a line of its own, rather than one tacked onto
// whatever move came last.
func (b *Board) end(outcome string) {
	b.action = ""
	b.conclude(outcome)
}

// key identifies a position for the repetition count: the pieces, who is to
// move, what may still castle and any square open to capture in passing.
func (b *Board) key() string {
	var sb strings.Builder
	sb.Grow(size*size + len(castleSquares) + 2)

	for row := range size {
		for col := range size {
			piece := b.pieces[row][col]
			if piece == 0 {
				piece = '.'
			}

			sb.WriteByte(piece)
		}
	}

	sb.WriteByte(byte(b.turn))

	for _, s := range castleSquares {
		if b.moved[s[0]][s[1]] {
			sb.WriteByte('-')
		} else {
			sb.WriteByte('+')
		}
	}

	return sb.String() + strconv.Itoa(b.passantCol)
}

// Tick takes d off the clock of the side to move, and ends the game if that
// runs them out.
func (b *Board) Tick(d time.Duration) {
	if b.over || !b.clock.On() {
		return
	}

	if !b.clock.Tick(b.turn, d) {
		b.over = true
		b.winner = b.turn.Other()
		b.setStatus(b.turn.String() + " ran out of time")

		return
	}

	b.refresh()
}

// setStatus shows what just happened and redraws the rest of the status with it.
func (b *Board) setStatus(action string) {
	b.action = action
	b.refresh()
}

// refresh rewrites the status: the clocks, what just happened, then whose turn
// it is. A Label splits on newlines, so they sit together with no gap.
func (b *Board) refresh() {
	lines := make([]string, 0, 3)

	if b.clock.On() {
		lines = append(lines, b.sideClock(White)+"   "+b.sideClock(Black))
	}

	lines = append(lines, b.action)

	switch {
	case b.over && b.drawn:
		lines = append(lines, "draw")
	case b.over:
		lines = append(lines, b.winner.String()+" wins")
	case b.inCheck(b.turn):
		lines = append(lines, b.turn.String()+" to move, in check")
	default:
		lines = append(lines, b.turn.String()+" to move")
	}

	b.status.Text = strings.Join(lines, "\n")
}

// sideClock is one side's time, bracketed like the cursor when it is running.
func (b *Board) sideClock(side Side) string {
	text := glyph(king(side)) + " " + clockText(b.clock.Left(side))

	if !b.over && b.turn == side {
		return "[" + text + "]"
	}

	return " " + text + " "
}

// king is the piece that labels a side's clock.
func king(side Side) byte {
	if side == White {
		return 'K'
	}

	return 'k'
}

// view is the board as a screen row: coordinates around the grid, and a redraw
// after every key so the selector keeps up with the cursor.
type view struct {
	*widgets.Container

	board *Board
}

func (v view) Select(cursor *stgui.Cursor, input stgui.Input) stgui.Action {
	action := v.Container.Select(cursor, input)

	v.board.curRow, v.board.curCol = cursor.Row, cursor.Col
	v.board.draw()

	return action
}

// RenderLines puts rank digits down the left and file letters along the bottom.
// They go on here rather than in the grid, so the cursor has only squares to
// walk.
func (v view) RenderLines() []string {
	w, h := v.board.square()

	ranks := v.Grid.RenderLines()
	out := make([]string, 0, len(ranks)+1)

	for i, rank := range ranks {
		// The digit goes on the line the pieces of that rank sit on.
		label := " "
		if i%h == (h-1)/2 {
			label = strconv.Itoa(size - i/h)
		}

		out = append(out, styleCoord+label+reset+rank)
	}

	return append(out, filesLine(w))
}

// filesLine is the letters along the bottom, each under the middle of its
// column rather than a column off it.
func filesLine(w int) string {
	var sb strings.Builder
	sb.WriteString(styleCoord)

	for col := range size {
		// One column for the rank digit, then the middle of this square.
		for sb.Len()-len(styleCoord) < 1+col*w+(w-1)/2 {
			sb.WriteByte(' ')
		}

		sb.WriteByte(byte('a' + col))
	}

	sb.WriteString(reset)

	return sb.String()
}

// CellAt shifts a click past the rank digits down the left, so a square is hit
// where it looks like it is.
func (v view) CellAt(col, row int) (int, int, bool) {
	return v.Grid.CellAt(col-1, row)
}

func (v view) Render() string {
	return strings.Join(v.RenderLines(), "\n")
}

// square is how big one square is drawn: columns then rows.
func (b *Board) square() (int, int) {
	return squareSizes[b.size][0], squareSizes[b.size][1]
}

// Resize draws the board a step bigger or smaller, stopping at the ends.
func (b *Board) Resize(step int) {
	next := b.size + step
	if next < 0 || next >= len(squareSizes) {
		return
	}

	b.size = next

	w, _ := b.square()
	b.grid.Style.MinCellWidth = w
	b.grid.Style.NoValue = strings.Repeat(" ", w)

	b.draw()
}

// squareText is one square, h lines of w columns, with whatever stands on it
// centred. A dot stands in for a square the held piece could move to, brackets
// mark the cursor, and round brackets a piece that can be taken.
func squareText(piece byte, target, cursor bool, w, h int) string {
	glyph := " "
	switch {
	case piece != 0:
		glyph = string(glyphs[piece])
	case target:
		glyph = dot
	}

	middle := " " + glyph + " "
	switch {
	case cursor:
		middle = "[" + glyph + "]"
	case target && piece != 0:
		middle = "(" + glyph + ")"
	}

	// The middle is always three wide, so centre it in the square.
	left := (w - 3) / 2
	line := strings.Repeat(" ", left) + middle + strings.Repeat(" ", w-3-left)

	if h == 1 {
		return line
	}

	lines := make([]string, h)
	for i := range lines {
		lines[i] = strings.Repeat(" ", w)
	}
	lines[(h-1)/2] = line

	return strings.Join(lines, "\n")
}

// glyph is the piece as it's drawn, for the status line.
func glyph(piece byte) string {
	return string(glyphs[piece])
}

// square as a chess player would write it, "e4".
func square(row, col int) string {
	return fmt.Sprintf("%c%d", 'a'+col, size-row)
}
