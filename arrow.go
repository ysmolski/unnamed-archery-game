package main

import (
	"math"

	"github.com/faiface/pixel"
	"golang.org/x/image/colornames"
)

type Arrow struct {
	Entity
	baseScale      float64
	vel            pixel.Vec // velocity of the arrow
	target         pixel.Vec // where the arrow should drop down
	distance       float64   // half the distance from original spawn point to the target
	killSpotRadius float64
}

func NewArrow(spr *pixel.Sprite) *Arrow {
	a := &Arrow{Entity: *NewEntity(spr, pixel.ZV)}
	a.Deactivate()
	a.baseScale = 0.7
	a.ScaleXY = pixel.V(a.baseScale, a.baseScale)
	a.Color = colornames.Brown
	r := pixel.R(-1, -1, 1, 1)
	a.Collider = &r
	a.killSpotRadius = 6
	return a
}

func (a *Arrow) DistanceToTarget() float64 {
	return a.Pos.Sub(a.target).Len()
}

func (a *Arrow) CanKill() bool {
	// TODO: calculate this precisely?
	return a.DistanceToTarget() <= a.killSpotRadius
}

func (a *Arrow) Update() {
	if !a.Active {
		return
	}
	oldDist := a.DistanceToTarget()
	a.Pos = a.Pos.Add(a.vel.Scaled(engine.dt))
	newDist := a.DistanceToTarget()
	size := (a.distance - math.Abs(oldDist-a.distance)) / a.distance
	// scaling should depend on the a.distance
	a.ScaleXY.X = 0.5 + size*size
	a.ScaleXY.Y = 0.7 + size/2
	if newDist > oldDist {
		a.Active = false
		a.Visible = false
		return
	}
	acol := a.AbsCollider()
	walls := world.GetColliders(acol)
	for _, wall := range walls {
		if Collides(acol, wall) {
			a.Active = false
			a.Visible = false
			return
		}
	}
}
