package main

import (
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
)

type Camera struct {
	Window          *pixelgl.Window
	Pos             pixel.Vec
	Speed           float64
	Zoom, ZoomSpeed float64
}

func NewCamera(win *pixelgl.Window) *Camera {
	return &Camera{win, pixel.ZV, 1.0, 2, 1.1}
}

func (c *Camera) GetMatrix() pixel.Matrix {
	return pixel.IM.Scaled(c.Pos, c.Zoom).Moved(c.Window.Bounds().Center().Sub(c.Pos))
}

func (c *Camera) Follow(p pixel.Vec) {
	dist := p.Sub(c.Pos).Len()
	if dist > 32 {
		c.Pos = pixel.Lerp(c.Pos, p, c.Speed*engine.dt)
	}
}

func (c *Camera) Update() {
	// if c.Window.Pressed(pixelgl.KeyLeft) {
	// 	c.Pos.X -= c.Speed * engine.dt * (1.0 / c.Zoom)
	// }
	// if c.Window.Pressed(pixelgl.KeyRight) {
	// 	c.Pos.X += c.Speed * engine.dt * (1.0 / c.Zoom)
	// }
	// if c.Window.Pressed(pixelgl.KeyUp) {
	// 	c.Pos.Y -= c.Speed * engine.dt * (1.0 / c.Zoom)
	// }
	// if c.Window.Pressed(pixelgl.KeyDown) {
	// 	c.Pos.Y += c.Speed * engine.dt * (1.0 / c.Zoom)
	// }
	// c.Zoom *= math.Pow(c.ZoomSpeed, c.Window.MouseScroll().Y)
}
