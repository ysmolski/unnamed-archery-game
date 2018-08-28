package main

import (
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
)

type Hero struct {
	Entity
	vel      pixel.Vec
	maxVel   float64
	limitVel float64
	accel    float64
}

func NewHero(s *pixel.Sprite, pos pixel.Vec, maxVel, limitVel, accel float64) *Hero {
	e := NewEntity(s, pos)
	return &Hero{
		Entity:   *e,
		maxVel:   maxVel,
		limitVel: limitVel,
		accel:    accel,
	}
}

func (h *Hero) Update(walls []pixel.Rect) {
	dx := 0.0
	if engine.win.Pressed(pixelgl.KeyA) {
		dx = -h.accel * engine.dt
	} else if engine.win.Pressed(pixelgl.KeyD) {
		dx = +h.accel * engine.dt
	} else {
		// TODO: handle deceleration correctly, don't let it oscilate around 0.
		if h.vel.X > h.limitVel {
			dx = -h.accel * engine.dt
		} else if h.vel.X < -h.limitVel {
			dx = +h.accel * engine.dt
		} else {
			h.vel.X = 0
		}
	}
	h.vel.X = pixel.Clamp(h.vel.X+dx, -h.maxVel, h.maxVel)

	dy := 0.0
	if engine.win.Pressed(pixelgl.KeyS) {
		dy = -h.accel * engine.dt
	} else if engine.win.Pressed(pixelgl.KeyW) {
		dy = +h.accel * engine.dt
	} else {
		if h.vel.Y > h.limitVel {
			dy = -h.accel * engine.dt
		} else if h.vel.Y < -h.limitVel {
			dy = +h.accel * engine.dt
		} else {
			h.vel.Y = 0
		}
	}
	h.vel.Y = pixel.Clamp(h.vel.Y+dy, -h.maxVel, h.maxVel)

	// limit diagonal speed
	actualVel := h.vel.Len()
	if actualVel > h.maxVel {
		h.vel = h.vel.Scaled(h.maxVel / actualVel)
	}

	delta := h.vel.Scaled(engine.dt)
	colWorld := h.AbsCollider()
	c := colWorld.Moved(delta)
	for _, wall := range walls {
		if Intersects(c, wall) {
			// Try to zero movement on one of the axes and continue if there is no collision.
			tdelta := delta
			tdelta.Y = 0
			c = colWorld.Moved(tdelta)
			if !Intersects(c, wall) {
				h.vel.Y = 0
				delta = tdelta
				continue
			}
			tdelta = delta
			tdelta.X = 0
			c = colWorld.Moved(tdelta)
			if !Intersects(c, wall) {
				h.vel.X = 0
				delta = tdelta
				continue
			}
		}
		if delta == pixel.ZV {
			// bail when velocity is zero
			break
		}
	}
	h.Pos = h.Pos.Add(delta)
}
