package main

import (
	"fmt"
	"time"

	"github.com/faiface/pixel/pixelgl"
)

type Engine struct {
	elapsed          float64
	prevFrameStarted time.Time
	dt               float64
	win              *pixelgl.Window
	winCfg           *pixelgl.WindowConfig
	fpsCounter       int
	fpsTicker        <-chan time.Time
}

func NewEngine(cfg *pixelgl.WindowConfig) *Engine {
	win, err := pixelgl.NewWindow(*cfg)
	if err != nil {
		panic(err)
	}
	win.SetSmooth(false)

	return &Engine{
		win:              win,
		winCfg:           cfg,
		fpsTicker:        time.Tick(time.Second),
		prevFrameStarted: time.Now(),
	}
}

func (e *Engine) fpsHandler() {
	e.fpsCounter++
	select {
	case <-e.fpsTicker:
		e.win.SetTitle(fmt.Sprintf("%s | fps: %d", e.winCfg.Title, e.fpsCounter))
		e.fpsCounter = 0
	default:
	}
}
