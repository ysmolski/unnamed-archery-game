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

func Collides(a, b pixel.Rect) bool {
	x1 := math.Max(a.Min.X, b.Min.X)
	y1 := math.Max(a.Min.Y, b.Min.Y)
	x2 := math.Min(a.Max.X, b.Max.X)
	y2 := math.Min(a.Max.Y, b.Max.Y)

	if x1 > x2 || y1 > y2 {
		return false
	}
	return true
}

var (
	engine *Engine
	world  *World
)

var (
	darkblue = color.RGBA{0, 9, 26, 255}
	darkgray = color.RGBA{100, 111, 130, 255}
)

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

	trid := &pixel.TrianglesData{}
	batch := pixel.NewBatch(trid, tileset)
	imd := imdraw.New(nil)

	camera := NewCamera(engine.win)
	camera.Pos = pixel.V(216, 83)

	// font
	atlas := text.NewAtlas(basicfont.Face7x13, text.ASCII)
	mPosTxt := text.New(pixel.V(-32, -32), atlas)

	world = NewWorld(28, 14, sSize)
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
		pixel.V(200, 100),
		100,
		800,
	)
	r := pixel.R(-spr.Frame().W()/2.5, -spr.Frame().H()/2.5, spr.Frame().W()/2.5, spr.Frame().H()/3)
	hero.Collider = &r

	sprArrow := pixel.NewSprite(tileset, frames[26])
	arrow := NewArrow(sprArrow)

	sprSlime := pixel.NewSprite(tileset, frames[15])
	var slimes [100]*Slime
	for i := range slimes {
		slimes[i] = NewSlime(sprSlime)
	}
	activeSlimes := 0
	slimeTicker := time.Tick(4 * time.Second)

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

		arrow.Update()
		if win.JustPressed(pixelgl.MouseButton1) && !arrow.Active {
			arrow.Active = true
			arrow.Visible = true
			arrow.Pos = hero.Pos.Add(gunDir.Scaled(12))
			arrow.Angle = gunDir.Angle()
			arrow.vel = gunDir.Scaled(150).Add(hero.velocity.Scaled(0.01))
			arrow.target = mousePos
			arrow.distance = arrow.Pos.Sub(arrow.target).Len() / 2
		}

		for i := range slimes {
			if slimes[i].Active {
				slimes[i].Update(hero, arrow)
			}
		}

		select {
		case <-slimeTicker:
			// spawn new slime
			if activeSlimes >= len(slimes) {
				break
			}
			free := -1
			for i := range slimes {
				if !slimes[i].Active {
					free = i
					break
				}
			}
			if free == -1 {
				break
			}
			slimes[free].Spawn()
		default:
		}

		mPosTxt.Clear()
		fmt.Fprintf(mPosTxt, "mpos: %6.3f %6.3f\n", mousePos.X, mousePos.Y)
		fmt.Fprintf(mPosTxt, "hpos: %6.3f %6.3f\n", hero.Pos.X, hero.Pos.Y)
		fmt.Fprintf(mPosTxt, "hvel: %6.3f %6.3f %6.3f\n", hero.velocity.X, hero.velocity.Y, hero.velocity.Len())
		fmt.Fprintf(mPosTxt, "health: %6.3f\n", hero.health)

		//
		// draw
		///////////////////////////////////////////////
		win.Clear(darkblue)

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
			sprWall.DrawColorMask(batch, m, darkgray)
		}
		if arrow != nil {
			arrow.Draw(batch)
		}
		for i := range slimes {
			slimes[i].Draw(batch)
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
