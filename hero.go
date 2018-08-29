package main

import (
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"golang.org/x/image/colornames"
)

type Hero struct {
	Entity
	velocity          pixel.Vec
	maxVel            float64
	accel             float64
	health, maxHealth float64
}

func NewHero(s *pixel.Sprite, pos pixel.Vec, maxVel, accel float64) *Hero {
	e := NewEntity(s, pos)
	return &Hero{
		Entity:    *e,
		maxVel:    maxVel,
		accel:     accel,
		maxHealth: 100,
		health:    100,
	}
}

func (h *Hero) Damage(health float64) {
	h.health += health
}

func (h *Hero) Update(walls []pixel.Rect) {
	pct := h.health / h.maxHealth * 100
	switch {
	case pct >= 80:
		h.Color = colornames.White
	case pct >= 50:
		h.Color = colornames.Peachpuff
	case pct >= 25:
		h.Color = colornames.Rosybrown
	case pct > 15:
		h.Color = colornames.Brown
	case pct <= 0:
		h.Color = colornames.Red
	}

	daccel := h.accel * engine.dt

	dx := 0.0
	if engine.win.Pressed(pixelgl.KeyA) {
		dx = -daccel
	} else if engine.win.Pressed(pixelgl.KeyD) {
		dx = +daccel
	} else {
		// TODO: handle deceleration correctly, don't let it oscilate around 0.
		if h.velocity.X >= daccel {
			dx = -daccel
		} else if h.velocity.X <= -daccel {
			dx = +daccel
		} else {
			h.velocity.X = 0
		}
	}
	h.velocity.X = pixel.Clamp(h.velocity.X+dx, -h.maxVel, h.maxVel)

	dy := 0.0
	if engine.win.Pressed(pixelgl.KeyS) {
		dy = -daccel
	} else if engine.win.Pressed(pixelgl.KeyW) {
		dy = +daccel
	} else {
		if h.velocity.Y >= daccel {
			dy = -daccel
		} else if h.velocity.Y <= -daccel {
			dy = +daccel
		} else {
			h.velocity.Y = 0
		}
	}
	h.velocity.Y = pixel.Clamp(h.velocity.Y+dy, -h.maxVel, h.maxVel)

	// limit diagonal speed
	actualVel := h.velocity.Len()
	if actualVel > h.maxVel {
		h.velocity = h.velocity.Scaled(h.maxVel / actualVel)
	}

	delta := h.velocity.Scaled(engine.dt)
	colWorld := h.AbsCollider()
	c := colWorld.Moved(delta)
	for _, wall := range walls {
		if Collides(c, wall) {
			// Try to zero movement on one of the axes and continue if there is no collision.
			tdelta := delta
			tdelta.Y = 0
			c = colWorld.Moved(tdelta)
			if !Collides(c, wall) {
				h.velocity.Y = 0
				delta = tdelta
				continue
			}
			tdelta = delta
			tdelta.X = 0
			c = colWorld.Moved(tdelta)
			if !Collides(c, wall) {
				h.velocity.X = 0
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
