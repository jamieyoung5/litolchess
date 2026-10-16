package game

import (
	"maps"
	"time"
)

// snapshot is everything a move changes. Taking a copy is a lot simpler than
// working out how to undo each kind of move, and the position is only an array.
type snapshot struct {
	pieces [size][size]byte
	moved  [size][size]bool

	turn                   Side
	passantRow, passantCol int
	halfmoves              int
	repeats                int
	seen                   map[string]int

	action string
	over   bool
	drawn  bool
	winner Side
	offer  Side

	white, black time.Duration

	// next is what happened from here, so taking it back can say what it was.
	next string
}

// save records the position before something that can be taken back.
func (b *Board) save() {
	b.history = append(b.history, snapshot{
		pieces:     b.pieces,
		moved:      b.moved,
		turn:       b.turn,
		passantRow: b.passantRow,
		passantCol: b.passantCol,
		halfmoves:  b.halfmoves,
		repeats:    b.repeats,
		seen:       maps.Clone(b.seen),
		action:     b.action,
		over:       b.over,
		drawn:      b.drawn,
		winner:     b.winner,
		offer:      b.offer,
		white:      b.timeLeft(White),
		black:      b.timeLeft(Black),
	})
}

// describe names what was just done, so undo can report it.
func (b *Board) describe(what string) {
	if len(b.history) > 0 {
		b.history[len(b.history)-1].next = what
	}
}

// Undo takes back the last move, and keeps going while the side to move plays
// itself, so an engine's reply isn't left standing on its own.
func (b *Board) Undo() {
	undone, ok := b.undoOne()
	if !ok {
		b.setStatus("nothing to take back")
		return
	}

	for b.automatic != nil && b.automatic(b.turn) {
		if _, ok := b.undoOne(); !ok {
			break
		}
	}

	b.setStatus("took back " + undone)
	b.draw()
}

// undoOne restores the last snapshot and reports what it took back.
func (b *Board) undoOne() (string, bool) {
	if len(b.history) == 0 {
		return "", false
	}

	last := b.history[len(b.history)-1]
	b.history = b.history[:len(b.history)-1]

	b.pieces = last.pieces
	b.moved = last.moved
	b.turn = last.turn
	b.passantRow, b.passantCol = last.passantRow, last.passantCol
	b.halfmoves = last.halfmoves
	b.repeats = last.repeats
	b.seen = last.seen
	b.action = last.action
	b.over = last.over
	b.drawn = last.drawn
	b.winner = last.winner
	b.offer = last.offer

	if b.clock.On() {
		b.clock.white, b.clock.black = last.white, last.black
	}

	// Whatever was in hand went back on the board with everything else.
	b.holding = 0
	b.legal = nil

	return last.next, true
}

// resign hands the game to the other side.
func (b *Board) resign() {
	if b.over {
		return
	}

	b.save()
	b.winner = b.turn.Other()
	b.end(b.turn.String() + " resigns")
	b.describe("the resignation")
}

// proposeDraw offers a draw, or takes one that is already on the table.
func (b *Board) proposeDraw() {
	if b.over {
		return
	}

	if b.offer == b.turn.Other() {
		b.save()
		b.end("draw agreed")
		b.describe("the draw")

		return
	}

	b.offer = b.turn
	b.setStatus(b.turn.String() + " offers a draw")
}

func (b *Board) timeLeft(side Side) time.Duration {
	if !b.clock.On() {
		return 0
	}

	return b.clock.Left(side)
}
