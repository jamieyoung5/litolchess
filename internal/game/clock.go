package game

import (
	"fmt"
	"time"
)

// Clock is the time each side has left. A zero limit means an untimed game and
// nothing here counts.
type Clock struct {
	limit time.Duration
	white time.Duration
	black time.Duration
}

func NewClock(limit time.Duration) *Clock {
	return &Clock{limit: limit, white: limit, black: limit}
}

// On reports whether the game is timed.
func (c *Clock) On() bool {
	return c != nil && c.limit > 0
}

// Left is the time side has to play with.
func (c *Clock) Left(side Side) time.Duration {
	if side == White {
		return c.white
	}

	return c.black
}

// Tick takes d off side's time and reports whether any is left.
func (c *Clock) Tick(side Side, d time.Duration) bool {
	left := max(c.Left(side)-d, 0)

	if side == White {
		c.white = left
	} else {
		c.black = left
	}

	return left > 0
}

// clockText is a time as a player reads it, "4:07".
func clockText(d time.Duration) string {
	seconds := int(d.Seconds())
	return fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
}
