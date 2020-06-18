package main

import (
	"math"

	"github.com/faiface/pixel"
	"golang.org/x/image/colornames"
)

type Arrow struct {
	Entity
	StuckSprite  *pixel.Sprite
	baseScale    float64
	vel          pixel.Vec // velocity of the arrow
	target       pixel.Vec // where the arrow should drop down
	halfDistance float64   // half the distance from original spawn point to the target
	maxHeight    float64
	State        ArrowState
}

// Arrow starts this far from the center of the hero.
const ArrowStartDistance = 10.0

type ArrowState uint8

const (
	ArrowInactive ArrowState = iota
	ArrowQuiver
	ArrowHands
	ArrowFlying
	ArrowStuck
)

func NewArrow(normal, stuck *pixel.Sprite) *Arrow {
	a := &Arrow{Entity: *NewEntity(normal, pixel.ZV)}
	a.StuckSprite = stuck
	a.Deactivate()
	a.State = ArrowInactive
	a.baseScale = 1
	a.Color = colornames.Goldenrod
	r := pixel.R(-1, -1, 1, 1)
	a.Collider = &r
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

func (a *Arrow) ToHands() {
	a.Active = true
	a.Visible = true
	a.State = ArrowHands
}

func (a *Arrow) ToQuiver() {
	a.Active = true
	a.Visible = true
	a.State = ArrowQuiver
}

func (a *Arrow) AttachToHands(from, to pixel.Vec) {
	dir := to.Sub(from).Unit()
	a.Pos = from.Add(dir.Scaled(ArrowStartDistance))
	a.Angle = dir.Angle()
	a.ScaleXY.X = 1.0
	a.ScaleXY.Y = 1.0
}

func (a *Arrow) AttachToQuiver(pos pixel.Vec, idx int) {
	a.Pos = pos.Add(pixel.V(-8+3*float64(idx), 7))
	a.Angle = math.Pi / 2
	a.ScaleXY.X = 0.5
	a.ScaleXY.Y = 0.5
}

func (a *Arrow) Fly(from, to, relational pixel.Vec) {
	a.State = ArrowFlying
	dir := to.Sub(from).Unit()
	a.Pos = from.Add(dir.Scaled(ArrowStartDistance))
	a.Angle = dir.Angle()
	a.vel = dir.Scaled(150).Add(relational)
	a.target = to
	a.halfDistance = a.Pos.Sub(a.target).Len() / 2
	// height takes values in range [0, 50]
	a.maxHeight = pixel.Clamp(a.halfDistance/1.2, 0, 100)
	// fmt.Println(a.halfDistance, a.maxHeight)
}

func (a *Arrow) Kills(col pixel.Rect) bool {
	return a.State == ArrowFlying && a.CanKill() && collides(col, a.AbsCollider())
}

func (a *Arrow) Stick() {
	a.State = ArrowStuck
}

func (a *Arrow) Update() {
	if a.State == ArrowStuck {

	}

	if !a.Active || a.State != ArrowFlying {
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
	a.ScaleXY.X = a.baseScale + size*a.maxHeight/100 - perspect
	a.ScaleXY.Y = a.baseScale + size*a.maxHeight/100
	//fmt.Printf("%4.2f %4.2f\n", size, a.ScaleXY.X)
	if newDist > oldDist {
		// a.Active = false
		// a.Visible = false
		a.State = ArrowStuck
		return
	}
	acol := a.AbsCollider()
	walls := world.GetColliders(acol)
	for _, wall := range walls {
		if collides(acol, wall) {
			// a.Active = false
			// a.Visible = false
			a.State = ArrowStuck
			return
		}
	}
}

func (a *Arrow) Draw(t pixel.Target) {
	if a.Visible {
		m := pixel.IM.ScaledXY(pixel.ZV, a.ScaleXY).Rotated(pixel.ZV, a.Angle).Moved(a.Pos)
		if a.State == ArrowStuck {
			a.StuckSprite.DrawColorMask(t, m, a.Color)
		} else {
			a.Sprite.DrawColorMask(t, m, a.Color)
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
