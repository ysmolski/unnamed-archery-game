package main

import (
	"math"

	"github.com/faiface/pixel"
	"golang.org/x/image/colornames"
)

type Arrow struct {
	Entity
	baseScale    float64
	vel          pixel.Vec // velocity of the arrow
	target       pixel.Vec // where the arrow should drop down
	halfDistance float64   // half the distance from original spawn point to the target
	maxHeight    float64
	flying       bool
}

const ArrowStart = 10.0

func NewArrow(spr *pixel.Sprite) *Arrow {
	a := &Arrow{Entity: *NewEntity(spr, pixel.ZV)}
	a.Deactivate()
	a.baseScale = 0.7
	a.ScaleXY = pixel.V(a.baseScale, a.baseScale)
	a.Color = colornames.Goldenrod
	r := pixel.R(-1, -1, 1, 1)
	a.Collider = &r
	a.flying = false
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

func (a *Arrow) Spawn() {
	a.Active = true
	a.Visible = true
	a.flying = false
}

func (a *Arrow) SyncWith(from, to pixel.Vec) {
	dir := to.Sub(from).Unit()
	a.Pos = from.Add(dir.Scaled(ArrowStart))
	a.Angle = dir.Angle()
	a.ScaleXY.X = 1.0
	a.ScaleXY.Y = 1.0
}

func (a *Arrow) Fly(from, to, relational pixel.Vec) {
	a.flying = true
	dir := to.Sub(from).Unit()
	a.Pos = from.Add(dir.Scaled(ArrowStart))
	a.Angle = dir.Angle()
	a.vel = dir.Scaled(150).Add(relational)
	a.target = to
	a.halfDistance = a.Pos.Sub(a.target).Len() / 2
	// height takes values in range [0, 50]
	a.maxHeight = pixel.Clamp(a.halfDistance/1.2, 0, 100)
	// fmt.Println(a.halfDistance, a.maxHeight)
}

func (a *Arrow) Update() {
	if !a.Active || !a.flying {
		return
	}
	size := math.Sqrt(a.DistanceFromEnds())
	oldDist := a.DistanceToTarget()
	a.Pos = a.Pos.Add(a.vel.Scaled(engine.dt))
	newDist := a.DistanceToTarget()
	// Maximum scaling should depend on the a.distance.
	// If we shot on short distance then arrow should not rise high to the air.
	perspect := a.maxHeight / 150
	if newDist < a.halfDistance {
		// make size smaller close to the target since arrow drops to the floow
		perspect += (a.halfDistance - newDist) / a.halfDistance / 5
	}
	a.ScaleXY.X = 1.0 + size*a.maxHeight/100 - perspect
	a.ScaleXY.Y = 1.0 + size*a.maxHeight/100
	//fmt.Printf("%4.2f %4.2f\n", size, a.ScaleXY.X)
	if newDist > oldDist {
		a.Active = false
		a.Visible = false
		a.flying = false
		return
	}
	acol := a.AbsCollider()
	walls := world.GetColliders(acol)
	for _, wall := range walls {
		if Collides(acol, wall) {
			a.Active = false
			a.Visible = false
			a.flying = false
			return
		}
	}
}

func firstFreeArrow(s []*Arrow) int {
	free := -1
	for i := range s {
		if !s[i].Active {
			free = i
			break
		}
	}
	return free
}
