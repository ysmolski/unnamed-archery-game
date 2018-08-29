package main

import (
	"math/rand"

	"github.com/faiface/pixel"
	"golang.org/x/image/colornames"
)

type Slime struct {
	Entity
	speed     float64
	drainRate float64
	rotation  float64
}

func NewSlime(spr *pixel.Sprite) *Slime {
	sl := &Slime{Entity: *NewEntity(spr, pixel.ZV)}
	sl.ScaleXY = pixel.V(1, 1)
	sl.Color = colornames.Red
	s := float64(world.gridSize) / 2.5
	r := pixel.R(-s, -s, s, s)
	sl.Collider = &r
	sl.speed = 40
	sl.drainRate = 50
	sl.Active = false
	sl.Visible = false
	return sl
}

func (s *Slime) Spawn() {
	s.Pos.X = rand.Float64()*float64((world.width-3)*world.gridSize) + float64(world.gridSize)
	s.Pos.Y = rand.Float64()*float64((world.height-3)*world.gridSize) + float64(world.gridSize)
	s.rotation = rand.Float64()
	s.speed = s.rotation*40 + 30
	//s.speed /= 1000
	// TODO: check that we dont overlap the player
	s.Active = true
	s.Visible = true
}

func (s *Slime) Update(h *Hero, arrow *Arrow) {
	// Slimes should "see" the player and fly to touch the player.
	// TODO: Implement spiralled movement.
	dir := h.Pos.Sub(s.Pos).Unit().Scaled(s.speed * engine.dt)
	s.Pos = s.Pos.Add(dir)
	s.Angle += s.rotation * engine.dt

	wcol := s.AbsCollider()
	if Collides(wcol, h.AbsCollider()) {
		h.Damage(-s.drainRate * engine.dt)
		h.SlowDown()
	}

	if arrow.Active && arrow.CanKill() && Collides(wcol, arrow.AbsCollider()) {
		s.Deactivate()
		arrow.Deactivate()
	}
}
