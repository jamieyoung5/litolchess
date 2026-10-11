package game

import (
	"fmt"
	"os"
	"strings"

	"github.com/jamieyoung5/stgui"
	"github.com/jamieyoung5/stgui/widgets"
)

const (
	size = 8

	// Three columns leaves room for the selector's brackets either side of
	// the piece.
	squareWidth = 3

	// The cursor starts on e2.
	startRow, startCol = size - 2, 4

	reset = "\033[0m"
)

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

	// holding is zero when nothing is picked up.
	holding          byte
	fromRow, fromCol int

	curRow, curCol int
}

func NewBoard(startingPos [size]string) *Board {
	style := &stgui.GridStyle{
		NoValue:       strings.Repeat(" ", squareWidth),
		MinCellWidth:  squareWidth,
		SelectedStyle: styleCursor,
	}

	b := &Board{
		grid:   stgui.NewEmptyGrid(size, size, style),
		status: widgets.NewLabel("enter: pick up   q: quit"),
		curRow: startRow,
		curCol: startCol,
	}

	for row, rank := range startingPos {
		for col := range rank {
			if rank[col] != '.' {
				b.pieces[row][col] = rank[col]
			}
		}
	}

	b.draw()

	return b
}

func (b *Board) Display() {
	container := widgets.NewContainer(b.grid).OnKey(b.onKey)
	board := view{Container: container, board: b}

	screen := stgui.NewScreen(stgui.NewCursor(board, b.grid, b.curRow, b.curCol),
		stgui.Row(board),
		stgui.Row(b.status),
	)

	if err := stgui.NewApp(screen).QuitOn("q").Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

func (b *Board) draw() {
	b.grid.Each(func(row, col int, cell *stgui.Cell) {
		piece := b.pieces[row][col]

		// A piece in hand is off the board until it's put down, which marks
		// where it came from without a second style.
		if b.holding != 0 && row == b.fromRow && col == b.fromCol {
			piece = 0
		}

		b.grid.Set(row, col, squareText(piece, row == b.curRow && col == b.curCol))

		cell.Style = ""
		if (row+col)%2 == 1 {
			cell.Style = styleDarkSquare
		}
	})
}

func (b *Board) onKey(cursor *stgui.Cursor, input stgui.Input) stgui.Action {
	if !input.Is(stgui.ControlSelect) {
		return stgui.Stay()
	}

	row, col := cursor.Row, cursor.Col

	switch {
	case b.holding == 0:
		if b.pieces[row][col] == 0 {
			b.status.Text = fmt.Sprintf("%s is empty", square(row, col))
			break
		}

		b.holding = b.pieces[row][col]
		b.fromRow, b.fromCol = row, col
		b.status.Text = fmt.Sprintf("holding %s from %s", glyph(b.holding), square(row, col))

	case row == b.fromRow && col == b.fromCol:
		b.status.Text = fmt.Sprintf("put %s back on %s", glyph(b.holding), square(row, col))
		b.holding = 0

	default:
		taken := b.pieces[row][col]
		b.pieces[b.fromRow][b.fromCol] = 0
		b.pieces[row][col] = b.holding

		if taken != 0 {
			b.status.Text = fmt.Sprintf("%s takes %s on %s", glyph(b.holding), glyph(taken), square(row, col))
		} else {
			b.status.Text = fmt.Sprintf("%s to %s", glyph(b.holding), square(row, col))
		}

		b.holding = 0
	}

	return stgui.Stay()
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
	ranks := v.Grid.RenderLines()
	out := make([]string, 0, len(ranks)+1)

	for i, rank := range ranks {
		out = append(out, fmt.Sprintf("%s%d%s%s", styleCoord, size-i, reset, rank))
	}

	var files strings.Builder
	files.WriteString(" " + styleCoord)
	for col := range size {
		files.WriteString("  ")
		files.WriteByte(byte('a' + col))
	}
	files.WriteString(reset)

	return append(out, files.String())
}

func (v view) Render() string {
	return strings.Join(v.RenderLines(), "\n")
}

// squareText is a square's three columns: the piece, bracketed when the cursor
// is on it.
func squareText(piece byte, cursor bool) string {
	glyph := " "
	if piece != 0 {
		glyph = string(glyphs[piece])
	}

	if cursor {
		return "[" + glyph + "]"
	}

	return " " + glyph + " "
}

// glyph is the piece as it's drawn, for the status line.
func glyph(piece byte) string {
	return string(glyphs[piece])
}

// square as a chess player would write it, "e4".
func square(row, col int) string {
	return fmt.Sprintf("%c%d", 'a'+col, size-row)
}
