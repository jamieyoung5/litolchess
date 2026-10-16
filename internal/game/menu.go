package game

import (
	"fmt"
	"time"

	"github.com/jamieyoung5/stgui"
	"github.com/jamieyoung5/stgui/widgets"
)

// timeControls are the games on offer, in menu order. A zero limit is untimed.
var timeControls = []time.Duration{
	0,
	3 * time.Minute,
	5 * time.Minute,
	10 * time.Minute,
}

// menu is the opening screen. Picking a time control is what starts the game,
// so there is nothing else to confirm.
func menu(start func(limit time.Duration) stgui.Action) *stgui.Screen {
	rows := make([][]any, len(timeControls))
	for i, limit := range timeControls {
		rows[i] = []any{widgets.NewActionButton(timeControlName(limit), func() stgui.Action {
			return start(limit)
		})}
	}

	// Without a style of its own the grid falls back to stgui's own highlight,
	// which is the one bit of colour in the whole game.
	options := widgets.NewContainer(stgui.NewGrid(rows, &stgui.GridStyle{
		SelectedStyle: stgui.StyleReverse,
	}))

	return stgui.NewScreen(options.Focus(0, 0),
		stgui.Row(widgets.NewLabel("littolchess")),
		stgui.Row(options),
		stgui.Row(widgets.NewLabel("enter: start   q: quit")),
	)
}

// promotionMenu asks what a pawn on the far rank becomes.
func promotionMenu(side Side, choose func(piece byte) stgui.Action) *stgui.Screen {
	kinds := []byte{'Q', 'R', 'B', 'N'}

	row := make([]any, len(kinds))
	for i, k := range kinds {
		row[i] = widgets.NewActionButton(glyph(pieceOf(side, k)), func() stgui.Action {
			return choose(k)
		})
	}

	options := widgets.NewContainer(stgui.NewGrid([][]any{row}, &stgui.GridStyle{
		VerticalDivider: " ",
		SelectedStyle:   stgui.StyleReverse,
	}))

	return stgui.NewScreen(options.Focus(0, 0),
		stgui.Row(widgets.NewLabel("promote to")),
		stgui.Row(options),
	)
}

func timeControlName(limit time.Duration) string {
	if limit == 0 {
		return "no timer"
	}

	return fmt.Sprintf("%d min", int(limit.Minutes()))
}
