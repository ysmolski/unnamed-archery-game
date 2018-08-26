package main

import (
	"fmt"
	"image"
	"math"
	"os"
	"time"

	"image/color"
	_ "image/png"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/imdraw"
	"github.com/faiface/pixel/pixelgl"
	"golang.org/x/image/colornames"
)

func loadPicture(path string) (pixel.Picture, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	return pixel.PictureDataFromImage(img), nil
}

type Camera struct {
	Window          *pixelgl.Window
	Pos             pixel.Vec
	Speed           float64
	Zoom, ZoomSpeed float64
}

func NewCamera(win *pixelgl.Window) *Camera {
	return &Camera{win, pixel.ZV, 500.0, 2.0, 1.1}
}

func (c *Camera) GetMatrix() pixel.Matrix {
	return pixel.IM.Scaled(c.Pos, c.Zoom).Moved(c.Window.Bounds().Center().Sub(c.Pos))
}

func (c *Camera) Update(dt float64) {
	// if c.Window.Pressed(pixelgl.KeyA) {
	// 	c.Pos.X -= c.Speed * dt * (1.0 / c.Zoom)
	// }
	// if c.Window.Pressed(pixelgl.KeyD) {
	// 	c.Pos.X += c.Speed * dt * (1.0 / c.Zoom)
	// }
	// if c.Window.Pressed(pixelgl.KeyW) {
	// 	c.Pos.Y -= c.Speed * dt * (1.0 / c.Zoom)
	// }
	// if c.Window.Pressed(pixelgl.KeyS) {
	// 	c.Pos.Y += c.Speed * dt * (1.0 / c.Zoom)
	// }
	c.Zoom *= math.Pow(c.ZoomSpeed, c.Window.MouseScroll().Y)
}

type Entity struct {
	s        *pixel.Sprite
	mat      pixel.Matrix
	collider pixel.Rect
	color    color.Color
	vel      pixel.Vec
	maxVel   float64
	limitVel float64
	accel    float64
}

func NewEntity(s *pixel.Sprite, scale float64, pos pixel.Vec, collider pixel.Rect, color color.Color, maxVel, limitVel, accel float64) *Entity {
	mat := pixel.IM.Scaled(pixel.ZV, scale).Moved(pos)
	collider = collider.Moved(pos)
	return &Entity{
		s:        s,
		mat:      mat,
		collider: collider,
		color:    color,
		maxVel:   maxVel,
		limitVel: limitVel,
		accel:    accel,
	}
}

func (h *Entity) Update(win *pixelgl.Window, dt float64, wall pixel.Rect) {
	// oldV := h.vel

	dx := 0.0
	if win.Pressed(pixelgl.KeyA) {
		dx = -h.accel * dt
	} else if win.Pressed(pixelgl.KeyD) {
		dx = +h.accel * dt
	} else {
		if h.vel.X > h.limitVel {
			dx = -h.accel * dt
		} else if h.vel.X < -h.limitVel {
			dx = +h.accel * dt
		} else {
			h.vel.X = 0
		}
	}
	h.vel.X = pixel.Clamp(h.vel.X+dx, -h.maxVel, h.maxVel)

	dy := 0.0
	if win.Pressed(pixelgl.KeyS) {
		dy = -h.accel * dt
	} else if win.Pressed(pixelgl.KeyW) {
		dy = +h.accel * dt
	} else {
		if h.vel.Y > h.limitVel {
			dy = -h.accel * dt
		} else if h.vel.Y < -h.limitVel {
			dy = +h.accel * dt
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

	delta := h.vel.Scaled(dt)
	c := h.collider.Moved(delta)
	overlap := c.Intersect(wall)
	if overlap.W() > 0 {
		h.vel.X = 0
		delta.X = 0
	}
	if overlap.H() > 0 {
		h.vel.Y = 0
		delta.Y = 0
	}
	h.collider = h.collider.Moved(delta)
	h.mat = h.mat.Moved(delta)

	// if h.vel != oldV {
	// 	fmt.Println(h.vel)
	// }
}

func run() {
	// load tileset
	tileset, err := loadPicture("tileset.png")
	if err != nil {
		panic(err)
	}
	const sDim = 16
	var frames []pixel.Rect
	{
		b := tileset.Bounds()
		for y := b.Max.Y; y > b.Min.Y; y -= sDim {
			for x := b.Min.X; x < b.Max.X; x += sDim {
				frames = append(frames, pixel.R(x, y-sDim, x+sDim, y))
			}
		}
	}
	fmt.Printf("%v tiles loaded\n", len(frames))

	cfg := pixelgl.WindowConfig{
		Title:  "A World",
		Bounds: pixel.R(0, 0, 1000, 600),
		VSync:  false,
	}
	win, err := pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}
	win.SetSmooth(false)

	// fps
	last := time.Now()
	elapsedFrames := 0
	everySecond := time.Tick(time.Second)

	camera := NewCamera(win)

	spr := pixel.NewSprite(tileset, frames[2])
	hero := NewEntity(
		spr,
		1.0,
		pixel.V(-100, -100),
		pixel.R(-spr.Frame().W()/2, -spr.Frame().H()/2, spr.Frame().W()/2, spr.Frame().H()/2),
		colornames.White, 50, 1, 300,
	)

	wall := pixel.R(0, 0, 100, 100)

	trid := &pixel.TrianglesData{}
	batch := pixel.NewBatch(trid, tileset)
	imd := imdraw.New(nil)

	for !win.Closed() {
		dt := time.Since(last).Seconds()
		last = time.Now()

		camMat := camera.GetMatrix()
		win.SetMatrix(camMat)
		camera.Update(dt)

		hero.Update(win, dt, wall)

		wPos := camMat.Unproject(win.MousePosition())

		win.Clear(colornames.Forestgreen)

		// debug
		imd.Clear()
		imd.Color = colornames.Blueviolet
		drawRect(imd, hero.collider)
		drawRect(imd, wall)

		origin := hero.mat.Project(pixel.ZV)
		target := origin.Add(wPos.Sub(origin).Unit().Scaled(16))
		fmt.Println(origin, target)
		imd.Push(origin, target)
		imd.Line(1)

		imd.Draw(win)

		batch.Clear()
		hero.s.DrawColorMask(batch, hero.mat, hero.color)
		batch.Draw(win)
		win.Update()

		elapsedFrames++
		select {
		case <-everySecond:
			win.SetTitle(fmt.Sprintf("%s | fps: %d", cfg.Title, elapsedFrames))
			elapsedFrames = 0
		default:
		}
	}
}

func main() {
	pixelgl.Run(run)
}

func drawRect(imd *imdraw.IMDraw, r pixel.Rect) {
	a := r.Min
	b := pixel.V(r.Min.X, r.Max.Y)
	c := r.Max
	d := pixel.V(r.Max.X, r.Min.Y)
	imd.Push(a, b, c, d, a)
	imd.Line(0.5)
}
