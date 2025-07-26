package dialog

import (
	"math/rand/v2"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	yes = "Yes"
	no  = "No"
)

func New(promptMsg string, confirm, cancel func()) *tview.Modal {
	modal := tview.NewModal().
		SetText(promptMsg).
		AddButtons(dialogButtons()).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			switch buttonLabel {
			case yes:
				confirm()
			case no:
				cancel()
			}
		})
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			cancel()
			return nil
		}
		return event
	})
	return modal
}

func dialogButtons() []string {
	buttons := []string{no, no, no, yes}
	shuffleSlice(buttons[1:])
	return buttons
}

func shuffleSlice[T any](slice []T) {
	rand.Shuffle(len(slice), func(i, j int) {
		slice[i], slice[j] = slice[j], slice[i]
	})
}
