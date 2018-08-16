// mouse displays a text box and tests mouse interaction.  As you click
// and drag, boxes are displayed on screen.  Other events are reported in
// the box.  Press ESC twice to exit the program.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/encoding"
)

var defStyle tcell.Style

type game struct {
}

type player struct {
	x, y      uint
	moveDelay time.Duration
	timeToPos time.Duration
}

func main() {

	encoding.Register()

	s, e := tcell.NewScreen()
	if e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}
	if e := s.Init(); e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}
	defStyle = tcell.StyleDefault.
		Background(tcell.ColorBlack).
		Foreground(tcell.ColorWhite)
	s.SetStyle(defStyle)
	// s.EnableMouse()
	s.Clear()

	white := tcell.StyleDefault.
		Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	red := tcell.StyleDefault.Background(tcell.ColorRed)

	// launch polling goroutine
	evch := make(chan tcell.Event, 2)
	go func() {
		for {
			ev := s.PollEvent()
			if ev == nil {
				return
			}
			evch <- ev
		}
	}()

	pl := player{5, 5, 200 * time.Millisecond, 0}

	w, h := s.Size()
	var elapsed time.Duration

mainloop:
	for {
		started := time.Now()

		w, h = s.Size()

		select {
		case ev := <-evch:

			switch ev := ev.(type) {
			case *tcell.EventResize:
				s.Sync()
				s.SetContent(w-1, h-1, 'R', nil, red)

			case *tcell.EventKey:
				s.SetContent(w-1, h-2, ev.Rune(), nil, red)
				s.SetContent(w-1, h-1, 'K', nil, red)
				if ev.Key() == tcell.KeyEscape {
					s.Fini()
					break mainloop
				}
				evn := fmt.Sprintf("%19v", ev.Name())
				emitStr(s, w-len(evn), h-3, white, evn)

				key := ev.Rune()
				if key == 'w' {
					pl.y--
				}
				if key == 's' {
					pl.y++
				}
				if key == 'a' {
					pl.x--
				}
				if key == 'd' {
					pl.x++
				}
			default:
				s.SetContent(w-1, h-1, 'X', nil, red)
			}

		case <-time.After(33 * time.Millisecond):
			// no event happened, let's just wait a while
		}
		drawBox(s, 0, 0, w-20, h-2, white, ' ')
		// Previous frame render time
		emitStr(s, w-6, 1, white, fmt.Sprintf("%4vms", elapsed.Nanoseconds()/1000000))
		s.SetContent(int(pl.x), int(pl.y), '@', nil, white)
		s.Show()
		elapsed = time.Since(started)
	}
}

func emitStr(s tcell.Screen, x, y int, style tcell.Style, str string) {
	for _, c := range str {
		s.SetContent(x, y, c, nil, style)
		x++
	}
}

func drawBox(s tcell.Screen, x1, y1, x2, y2 int, style tcell.Style, r rune) {
	if y2 < y1 {
		y1, y2 = y2, y1
	}
	if x2 < x1 {
		x1, x2 = x2, x1
	}

	for col := x1; col <= x2; col++ {
		s.SetContent(col, y1, tcell.RuneHLine, nil, style)
		s.SetContent(col, y2, tcell.RuneHLine, nil, style)
	}
	for row := y1 + 1; row < y2; row++ {
		s.SetContent(x1, row, tcell.RuneVLine, nil, style)
		s.SetContent(x2, row, tcell.RuneVLine, nil, style)
	}
	if y1 != y2 && x1 != x2 {
		// Only add corners if we need to
		s.SetContent(x1, y1, tcell.RuneULCorner, nil, style)
		s.SetContent(x2, y1, tcell.RuneURCorner, nil, style)
		s.SetContent(x1, y2, tcell.RuneLLCorner, nil, style)
		s.SetContent(x2, y2, tcell.RuneLRCorner, nil, style)
	}
	for row := y1 + 1; row < y2; row++ {
		for col := x1 + 1; col < x2; col++ {
			s.SetContent(col, row, r, nil, style)
		}
	}
}

func drawSelect(s tcell.Screen, x1, y1, x2, y2 int, sel bool) {

	if y2 < y1 {
		y1, y2 = y2, y1
	}
	if x2 < x1 {
		x1, x2 = x2, x1
	}

	for row := y1; row <= y2; row++ {
		for col := x1; col <= x2; col++ {
			mainc, combc, style, width := s.GetContent(col, row)
			if style == tcell.StyleDefault {
				style = defStyle
			}
			style = style.Reverse(sel)
			s.SetContent(col, row, mainc, combc, style)
			col += width - 1
		}
	}
}
