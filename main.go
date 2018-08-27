package main

import (
	"flag"
	"fmt"
	"image"
	"log"
	"math"
	"math/rand"
	"os"
	"runtime/pprof"
	"time"

	"image/color"
	_ "image/png"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/imdraw"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font/basicfont"
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

func drawRect(imd *imdraw.IMDraw, r pixel.Rect) {
	a := r.Min
	b := pixel.V(r.Min.X, r.Max.Y)
	c := r.Max
	d := pixel.V(r.Max.X, r.Min.Y)
	imd.Push(a, b, c, d, a)
	imd.Line(1)
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
	if c.Window.Pressed(pixelgl.KeyLeft) {
		c.Pos.X -= c.Speed * dt * (1.0 / c.Zoom)
	}
	if c.Window.Pressed(pixelgl.KeyRight) {
		c.Pos.X += c.Speed * dt * (1.0 / c.Zoom)
	}
	if c.Window.Pressed(pixelgl.KeyUp) {
		c.Pos.Y -= c.Speed * dt * (1.0 / c.Zoom)
	}
	if c.Window.Pressed(pixelgl.KeyDown) {
		c.Pos.Y += c.Speed * dt * (1.0 / c.Zoom)
	}
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

func (h *Entity) Update(win *pixelgl.Window, dt float64, walls []pixel.Rect) {
	oldV := h.vel

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
	for _, wall := range walls {
		overlap := c.Intersect(wall)
		if overlap.H() > 0 {
			h.vel.Y = 0
			delta.Y = 0
			c = h.collider.Moved(delta)
			overlap = c.Intersect(wall)
		}
		if overlap.W() > 0 {
			h.vel.X = 0
			delta.X = 0
		}
		if delta == pixel.ZV {
			break
		}
	}
	h.collider = h.collider.Moved(delta)
	h.mat = h.mat.Moved(delta)

	if h.vel != oldV {
		fmt.Println(h.vel)
	}
}

type CellType uint8

const (
	CellEmpty = iota
	CellWall
	CellStone
	numberOfCellTypes
)

type World struct {
	gridSize      int // the side of one grid element
	width, height int
	cells         [][]CellType
}

func NewWorld(width, height, gridSize int) *World {
	w := &World{gridSize: gridSize, width: width, height: height}
	w.cells = make([][]CellType, width)
	for i := 0; i < width; i++ {
		w.cells[i] = make([]CellType, height)
		w.cells[i][0] = CellWall
		w.cells[i][height-1] = CellWall
		if i == 0 || i == width-1 {
			for j := 1; j < height-1; j++ {
				w.cells[i][j] = CellWall
			}
		}
	}
	// // random walls
	// for i := 0; i < 10; i++ {
	// 	x := rand.Intn(width)
	// 	y := rand.Intn(height)
	// 	w.cells[x][y] = CellWall
	// }
	return w
}

func (w *World) spaceToGrid(a float64) int {
	return int(a) / w.gridSize
}

func (w *World) GetColliders(a, b, c, d int) []pixel.Rect {
	var r []pixel.Rect
	halfSize := float64(w.gridSize / 2)
	for x := a; x <= c; x++ {
		for y := b; y <= d; y++ {
			if w.cells[x][y] != CellEmpty {
				r = append(r, pixel.R(
					float64(x*w.gridSize)-halfSize,
					float64(y*w.gridSize)-halfSize,
					float64(x*w.gridSize)+halfSize,
					float64(y*w.gridSize)+halfSize,
				))
			}
		}
	}
	return r
}

func run() {
	rand.Seed(int64(time.Now().Nanosecond()))
	// load tileset
	tileset, err := loadPicture("tileset.png")
	if err != nil {
		panic(err)
	}
	const sSize = 16
	var frames []pixel.Rect
	{
		b := tileset.Bounds()
		for y := b.Max.Y; y > b.Min.Y; y -= sSize {
			for x := b.Min.X; x < b.Max.X; x += sSize {
				frames = append(frames, pixel.R(x, y-sSize, x+sSize, y))
			}
		}
	}
	fmt.Printf("%v tiles loaded\n", len(frames))

	cfg := pixelgl.WindowConfig{
		Title:  "A World",
		Bounds: pixel.R(0, 0, 1000, 600),
		VSync:  true,
	}
	win, err := pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}
	win.SetSmooth(false)

	w := NewWorld(28, 14, sSize)
	sprWall := pixel.NewSprite(tileset, frames[256-37])
	matWalls := make([]pixel.Matrix, 0, 32*16)
	for x := 0; x < w.width; x++ {
		for y := 0; y < w.height; y++ {
			if w.cells[x][y] == CellWall {
				matWalls = append(matWalls, pixel.IM.Moved(pixel.V(float64(x*w.gridSize), float64(y*w.gridSize))))
			}
		}
	}

	spr := pixel.NewSprite(tileset, frames[2])
	hero := NewEntity(
		spr,
		1.0,
		pixel.V(16, 16),
		pixel.R(-spr.Frame().W()/2.5, -spr.Frame().H()/2.5, spr.Frame().W()/2.5, spr.Frame().H()/3),
		colornames.White, 50, 1, 300,
	)

	camera := NewCamera(win)
	camera.Pos = pixel.V(216, 83)

	trid := &pixel.TrianglesData{}
	batch := pixel.NewBatch(trid, tileset)
	imd := imdraw.New(nil)

	// font
	atlas := text.NewAtlas(basicfont.Face7x13, text.ASCII)
	mPosTxt := text.New(pixel.V(-32, -32), atlas)

	// fps
	last := time.Now()
	elapsedFrames := 0
	everySecond := time.Tick(time.Second)

	for !win.Closed() {
		dt := time.Since(last).Seconds()
		last = time.Now()

		camMat := camera.GetMatrix()
		win.SetMatrix(camMat)
		camera.Update(dt)

		a := (w.spaceToGrid(hero.collider.Min.X))
		b := (w.spaceToGrid(hero.collider.Min.Y))
		c := (w.spaceToGrid(hero.collider.Max.X) + 1)
		d := (w.spaceToGrid(hero.collider.Max.Y) + 1)
		walls := w.GetColliders(a, b, c, d)

		hero.Update(win, dt, walls)

		wPos := camMat.Unproject(win.MousePosition())
		mPosTxt.Clear()
		fmt.Fprintf(mPosTxt, "wpos: %6.3f %6.3f", wPos.X, wPos.Y)

		// grid := pixel.R(
		// 	float64(a*w.gridSize),
		// 	float64(b*w.gridSize),
		// 	float64(c*w.gridSize),
		// 	float64(d*w.gridSize),
		// )
		fmt.Fprintf(mPosTxt, "\ngrid: %v %v %v %v", a, b, c, d)

		//
		// draw
		//
		win.Clear(colornames.Forestgreen)
		// debug
		imd.Clear()
		imd.Color = colornames.Blueviolet
		drawRect(imd, hero.collider)
		// drawRect(imd, grid)

		origin := hero.mat.Project(pixel.ZV)

		// gun
		target := origin.Add(wPos.Sub(origin).Unit().Scaled(12))
		imd.Push(origin, target)
		imd.Line(2)

		imd.Draw(win)

		// tileset batch
		batch.Clear()
		hero.s.DrawColorMask(batch, hero.mat, hero.color)
		for _, m := range matWalls {
			sprWall.DrawColorMask(batch, m, colornames.White)
		}
		batch.Draw(win)

		// debug text
		mPosTxt.Draw(win, pixel.IM.Scaled(mPosTxt.Orig, .5))

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

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	pixelgl.Run(run)
}
