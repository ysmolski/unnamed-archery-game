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
	halfDistance   float64   // half the distance from original spawn point to the target
	maxHeight      float64
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
	return a.CurrentHeight() < 8
}

// DistanceFromEnds returns values in range 0 ... 1 ... 0,
// it returns 1 in the middle (the highest point) of trajectory.
func (a *Arrow) DistanceFromEnds() float64 {
	return (a.halfDistance - math.Abs(a.DistanceToTarget()-a.halfDistance)) / a.halfDistance
}

func (a *Arrow) CurrentHeight() float64 {
	return a.maxHeight * math.Sqrt(a.DistanceFromEnds())
}

func (a *Arrow) Spawn(from, to, relational pixel.Vec) {
	a.Active = true
	a.Visible = true
	dir := to.Sub(from).Unit()
	a.Pos = from.Add(dir.Scaled(6))
	a.Angle = dir.Angle()
	a.vel = dir.Scaled(150).Add(relational)
	a.target = to
	a.halfDistance = a.Pos.Sub(a.target).Len() / 2
	// height takes values in range [0, 50]
	a.maxHeight = pixel.Clamp(a.halfDistance/1.2, 0, 100)
	// fmt.Println(a.halfDistance, a.maxHeight)
}

func (a *Arrow) Update() {
	if !a.Active {
		return
	}
	size := math.Sqrt(a.DistanceFromEnds())
	oldDist := a.DistanceToTarget()
	a.Pos = a.Pos.Add(a.vel.Scaled(engine.dt))
	newDist := a.DistanceToTarget()
	// Maximum scaling should depend on the a.distance.
	// If we shot on short distance then arrow should not rise high to the air.
	a.ScaleXY.X = 1.0 + size*a.maxHeight/100 - a.maxHeight/150
	a.ScaleXY.Y = 1.0 + size*a.maxHeight/100
	//fmt.Printf("%4.2f %4.2f\n", size, a.ScaleXY.X)
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
