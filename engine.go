package main

import (
	"time"

	"github.com/faiface/pixel/pixelgl"
)

type Engine struct {
	started          time.Time
	elapsed          float64
	prevFrameStarted time.Time
	dt               float64
	win              *pixelgl.Window
	winCfg           *pixelgl.WindowConfig
	// fpsCounter       int
	// fpsTicker        <-chan time.Time
}

func NewEngine(cfg *pixelgl.WindowConfig) *Engine {
	win, err := pixelgl.NewWindow(*cfg)
	if err != nil {
		panic(err)
	}
	win.SetSmooth(false)

	return &Engine{
		started:          time.Now(),
		win:              win,
		winCfg:           cfg,
		prevFrameStarted: time.Now(),
	}
}

func (e *Engine) fpsHandler() {
	e.elapsed = time.Since(e.started).Seconds()
}
