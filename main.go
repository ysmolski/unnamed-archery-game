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

func Intersects(a, b pixel.Rect) bool {
	x1 := math.Max(a.Min.X, b.Min.X)
	y1 := math.Max(a.Min.Y, b.Min.Y)
	x2 := math.Min(a.Max.X, b.Max.X)
	y2 := math.Min(a.Max.Y, b.Max.Y)

	if x1 > x2 || y1 > y2 {
		return false
	}
	return true
}

type Arrow struct {
	Entity
	vel      pixel.Vec
	target   pixel.Vec
	distance float64
}

var engine *Engine

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
	engine = NewEngine(&cfg)

	world := NewWorld(28, 14, sSize)
	sprWall := pixel.NewSprite(tileset, frames[256-37])
	matWalls := make([]pixel.Matrix, 0, 32*16)
	for x := 0; x < world.width; x++ {
		for y := 0; y < world.height; y++ {
			if world.cells[x][y] == CellWall {
				matWalls = append(matWalls, pixel.IM.Moved(pixel.V(float64(x*world.gridSize), float64(y*world.gridSize))))
			}
		}
	}

	spr := pixel.NewSprite(tileset, frames[1])
	hero := NewHero(
		spr,
		pixel.V(16, 16),
		100, 100/30, 800,
	)
	r := pixel.R(-spr.Frame().W()/2.5, -spr.Frame().H()/2.5, spr.Frame().W()/2.5, spr.Frame().H()/3)
	hero.Collider = &r

	camera := NewCamera(engine.win)
	camera.Pos = pixel.V(216, 83)

	trid := &pixel.TrianglesData{}
	batch := pixel.NewBatch(trid, tileset)
	imd := imdraw.New(nil)

	// font
	atlas := text.NewAtlas(basicfont.Face7x13, text.ASCII)
	mPosTxt := text.New(pixel.V(-32, -32), atlas)

	sprArrow := pixel.NewSprite(tileset, frames[26])
	var arrow *Arrow

	win := engine.win
	for !win.Closed() {
		engine.dt = time.Since(engine.prevFrameStarted).Seconds()
		engine.prevFrameStarted = time.Now()

		camMat := camera.GetMatrix()
		win.SetMatrix(camMat)
		camera.Update()

		walls := world.GetColliders(hero.AbsCollider())
		hero.Update(walls)

		mousePos := camMat.Unproject(win.MousePosition())
		// gun
		origin := hero.Pos
		gunDir := mousePos.Sub(origin).Unit()
		gunStart := origin.Add(gunDir.Scaled(6))
		gunEnd := origin.Add(gunDir.Scaled(12))

		if arrow != nil {
			distance := arrow.Pos.Sub(arrow.target).Len()
			arrow.Pos = arrow.Pos.Add(arrow.vel.Scaled(engine.dt))
			arrow.Color = colornames.Brown
			//size := (ar.distance - math.Abs(distance-ar.distance)) / ar.distance
			//ar.Scale = 0.5 + size*size

			acol := arrow.AbsCollider()
			walls := world.GetColliders(acol)

			destroyed := false
			if arrow.Pos.Sub(arrow.target).Len() > distance {
				destroyed = true
			}
			for _, wall := range walls {
				if Intersects(acol, wall) {
					destroyed = true
					break
				}
			}
			if destroyed {
				arrow = nil
			}

		}

		if win.JustPressed(pixelgl.MouseButton1) && arrow == nil {
			arrow = &Arrow{Entity: *NewEntity(sprArrow, hero.Pos)}
			arrow.Pos = arrow.Pos.Add(gunDir.Scaled(12))
			arrow.Scale = 0.7
			arrow.Angle = gunDir.Angle()
			arrow.vel = gunDir.Scaled(200)
			arrow.target = mousePos
			arrow.distance = arrow.Pos.Sub(arrow.target).Len() / 2
			r := pixel.R(-4, -4, 4, 4)
			arrow.Collider = &r
			fmt.Println(arrow.Pos)
			fmt.Println(mousePos)
		}

		mPosTxt.Clear()
		fmt.Fprintf(mPosTxt, "mpos: %6.3f %6.3f\n", mousePos.X, mousePos.Y)
		fmt.Fprintf(mPosTxt, "hpos: %6.3f %6.3f\n", hero.Pos.X, hero.Pos.Y)
		fmt.Fprintf(mPosTxt, "hvel: %6.3f %6.3f\n", hero.vel.X, hero.vel.Y)

		//
		// draw
		//
		win.Clear(colornames.Forestgreen)

		// debug
		imd.Clear()
		imd.Color = colornames.Blueviolet

		//drawRect(imd, hero.Collider.Moved(origin))

		imd.Push(gunStart, gunEnd)
		imd.Line(2)

		imd.Draw(win)

		// debug text
		mPosTxt.Draw(win, pixel.IM.Scaled(mPosTxt.Orig, .5))

		// tileset batch
		batch.Clear()
		hero.Draw(batch)
		for _, m := range matWalls {
			sprWall.DrawColorMask(batch, m, colornames.White)
		}
		if arrow != nil {
			arrow.Draw(batch)
		}
		batch.Draw(win)

		win.Update()

		engine.fpsHandler()
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
