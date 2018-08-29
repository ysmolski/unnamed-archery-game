package main

import (
	"image/color"

	"github.com/faiface/pixel"
	"golang.org/x/image/colornames"
)

type Entity struct {
	Sprite   *pixel.Sprite
	Pos      pixel.Vec
	ScaleXY  pixel.Vec
	Angle    float64
	Color    color.RGBA
	Collider *pixel.Rect
	Active   bool // whether it should be updated
	Visible  bool // whether it should be drawn
}

func (e *Entity) AbsCollider() pixel.Rect {
	return e.Collider.Moved(e.Pos)
}

func NewEntity(sprite *pixel.Sprite, pos pixel.Vec) *Entity {
	e := &Entity{
		Sprite:  sprite,
		Pos:     pos,
		ScaleXY: pixel.V(1, 1),
		Angle:   0,
		Color:   colornames.White,
		Active:  true,
		Visible: true,
	}
	return e
}

func (e *Entity) Draw(t pixel.Target) {
	if e.Visible {
		m := pixel.IM.ScaledXY(pixel.ZV, e.ScaleXY).Rotated(pixel.ZV, e.Angle).Moved(e.Pos)
		e.Sprite.DrawColorMask(t, m, e.Color)
	}
}

func (e *Entity) Deactivate() {
	e.Active = false
	e.Visible = false
}

func (e *Entity) Activate() {
	e.Active = true
	e.Visible = true
}
