package main

import (
	"math/rand"

	"github.com/faiface/pixel"
	"golang.org/x/image/colornames"
)

type Slime struct {
	Entity
	speed float64
}

func NewSlime(spr *pixel.Sprite) *Slime {
	sl := &Slime{Entity: *NewEntity(spr, pixel.ZV)}
	sl.Scale = 1
	sl.Color = colornames.Brown
	s := float64(world.gridSize / 2)
	r := pixel.R(-s, -s, s, s)
	sl.Collider = &r
	sl.speed = 40
	sl.Active = false
	sl.Visible = false
	return sl
}

func (s *Slime) Spawn() {
	s.Pos.X = rand.Float64()*float64((world.width-3)*world.gridSize) + float64(world.gridSize)
	s.Pos.Y = rand.Float64()*float64((world.height-3)*world.gridSize) + float64(world.gridSize)
	// TODO: check that we dont overlap the player
	s.Active = true
	s.Visible = true
}

func (s *Slime) Update(h *Hero) {
	// Slimes should "see" the player and try to fly to touch the player.
	// TODO: Implement spiralled movement.
	dir := h.Pos.Sub(s.Pos).Unit().Scaled(s.speed * engine.dt)
	s.Pos = s.Pos.Add(dir)
}
