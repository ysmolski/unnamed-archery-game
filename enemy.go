package main

import (
	"math/rand"

	"github.com/faiface/pixel"
	"golang.org/x/image/colornames"
)

type Slime struct {
	Entity
	speed          float64
	drainRate      float64
	rotation       float64
	fixedDirection pixel.Vec
	fixed          bool
}

func NewSlime(spr *pixel.Sprite) *Slime {
	sl := &Slime{Entity: *NewEntity(spr, pixel.ZV)}
	sl.ScaleXY = pixel.V(1, 1)
	sl.Color = colornames.Red
	s := float64(world.gridSize) / 2.5
	r := pixel.R(-s, -s, s, s)
	sl.Collider = &r
	sl.speed = 40
	sl.drainRate = 120
	sl.Active = false
	sl.Visible = false
	return sl
}

func (s *Slime) Spawn() {
	p := world.RandomVec()
	for hero.Pos.Sub(p).Len() < 64 {
		p = world.RandomVec()
	}
	s.Pos = p
	s.rotation = rand.Float64() + 0.2
	s.speed = s.rotation*40 + 30 + engine.elapsed/10
	// s.speed /= 1000
	s.Active = true
	s.Visible = true
}

func (s *Slime) Update(arrows []*Arrow) {
	// Slimes should "see" the player and fly to touch the player.
	// TODO: Implement spiralled movement.
	var dir pixel.Vec
	dir = hero.Pos.Sub(s.Pos).Unit()
	if s.fixed {
		// Slime sticks to some constant directing until it goes out of range.
		dir = s.fixedDirection.Add(dir.Scaled(0.5)).Unit()
	}
	delta := dir.Scaled(s.speed * engine.dt)
	s.Pos = s.Pos.Add(delta)
	s.Angle += (s.rotation + 0.2) * engine.dt

	wallCollided := false
	wcol := s.AbsCollider()
	if Collides(wcol, hero.AbsCollider()) {
		hero.Damage(-s.drainRate * engine.dt)
		hero.SlowDown(0.7)
	} else {
		colWorld := s.AbsCollider()
		walls := world.GetColliders(colWorld)
		c := colWorld.Moved(delta)
		for _, wall := range walls {
			if Collides(c, wall) {
				s.Pos = s.Pos.Sub(delta)
				wallCollided = true
				if s.fixed {
					s.fixed = false
				}
			}
		}
	}

	diff := hero.Pos.Sub(s.Pos).Len()
	if diff <= 92 && !wallCollided {
		if diff < 48 && !s.fixed {
			s.fixedDirection = dir
			s.fixed = true
		}
		rate := (42 - diff) / 92
		// Speed up when diff==92 and then slow down when diff < 32
		s.Pos = s.Pos.Sub(delta.Scaled(rate))
	} else if s.fixed {
		s.fixed = false
	}

	if diff < 92 {
		rate := (92 - diff) / 300
		s.Angle += rate
	}

	for _, arrow := range arrows {
		if arrow.Kills(wcol) {
			s.Deactivate()
			arrow.Stick()
		}
	}
}

func firstFreeSlime(s []*Slime) int {
	free := -1
	for i := range s {
		if !s[i].Active {
			free = i
			break
		}
	}
	return free
}
