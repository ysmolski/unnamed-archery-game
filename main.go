package main

import (
	"flag"
	"fmt"
	"image"
	"log"
	"math"
	"math/rand"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
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
	darkblue = color.RGBA{0, 18, 34, 255}
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
		VSync:  false,
	}
	engine = NewEngine(&cfg)

	trid := &pixel.TrianglesData{}
	batch := pixel.NewBatch(trid, tileset)

	tridBg := &pixel.TrianglesData{}
	batchBg := pixel.NewBatch(tridBg, tileset)

	imd := imdraw.New(nil)

	camera := NewCamera(engine.win)
	camera.Pos = pixel.V(216, 83)

	// font
	atlas := text.NewAtlas(basicfont.Face7x13, text.ASCII)
	mPosTxt := text.New(pixel.V(8, engine.win.Bounds().Max.Y-16), atlas)

	sprWall := pixel.NewSprite(tileset, frames[256-37])
	var sprBG []*pixel.Sprite
	sprBG = append(sprBG, pixel.NewSprite(tileset, frames[176]))
	sprBG = append(sprBG, pixel.NewSprite(tileset, frames[177]))
	sprBG = append(sprBG, pixel.NewSprite(tileset, frames[178]))
	sprBG = append(sprBG, pixel.NewSprite(tileset, frames[256-37]))
	sprBG = append(sprBG, pixel.NewSprite(tileset, frames[256-36]))
	world = NewWorld(60, 30, sSize, sprWall, sprBG, batchBg)

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

	targetFrameTime := 8000 * time.Microsecond

	gcOnFrame := 120
	gcFrame := 0
	var gcTime time.Duration

	var (
		dtMax       float64
		dtUpdate    float64
		dtDraw      float64
		dtUpdateMax float64
		dtDrawMax   float64
	)

	win := engine.win
	for !win.Closed() {
		gcFrame++
		if gcFrame >= gcOnFrame {
			// When I run this game on machine with 2 cores CPU,
			// I get stuttering when something runs in background. Is it related to
			// GC or just a problem of slow hardware?
			// When I do not invoke GC manually I get many slow frames.
			gcSt := time.Now()
			runtime.GC()
			gcTime = time.Since(gcSt)
			gcFrame = 0
			dtMax = 0
			dtUpdateMax = 0
			dtDrawMax = 0
		}
		for time.Since(engine.prevFrameStarted) < targetFrameTime {
			time.Sleep(200 * time.Microsecond)
		}
		engine.dt = time.Since(engine.prevFrameStarted).Seconds()
		engine.prevFrameStarted = time.Now()
		if engine.dt > dtMax {
			dtMax = engine.dt
		}

		dtUpdateSt := time.Now()
		camera.Update()
		camMat := camera.GetMatrix()

		walls := world.GetColliders(hero.AbsCollider())
		hero.Update(walls)
		camera.Follow(hero.Pos)

		mousePos := camMat.Unproject(win.MousePosition())
		// gun
		origin := hero.Pos
		gunDir := mousePos.Sub(origin).Unit()
		gunStart := origin.Add(gunDir.Scaled(6))
		gunEnd := origin.Add(gunDir.Scaled(12))

		arrow.Update()
		if win.JustPressed(pixelgl.MouseButton1) && !arrow.Active {
			arrow.Spawn(hero.Pos, mousePos, hero.velocity.Scaled(0.2))
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
		// fmt.Fprintf(mPosTxt, "mpos: %6.3f %6.3f\n", mousePos.X, mousePos.Y)
		// fmt.Fprintf(mPosTxt, "hpos: %6.3f %6.3f\n", hero.Pos.X, hero.Pos.Y)
		// fmt.Fprintf(mPosTxt, "hvel: %6.3f %6.3f %6.3f\n", hero.velocity.X, hero.velocity.Y, hero.velocity.Len())
		// fmt.Fprintf(mPosTxt, "health: %6.1f\n", hero.health)
		fmt.Fprintf(mPosTxt, "cur dt: %6.5f\n", engine.dt)
		fmt.Fprintf(mPosTxt, "max dt: %6.5f\n", dtMax)
		fmt.Fprintf(mPosTxt, "dt upd: %6.5f\n", dtUpdateMax)
		fmt.Fprintf(mPosTxt, "dt dra: %6.5f\n", dtDrawMax)
		fmt.Fprintf(mPosTxt, "gc: %7d\n", gcTime.Nanoseconds())

		dtUpdate = time.Since(dtUpdateSt).Seconds()
		if dtUpdate > dtUpdateMax {
			dtUpdateMax = dtUpdate
		}

		//
		// draw
		///////////////////////////////////////////////
		dtDrawSt := time.Now()
		win.SetMatrix(camMat)
		win.Clear(darkblue)

		// tileset batch
		batch.Clear()
		world.Draw(batch)
		if arrow != nil {
			arrow.Draw(batch)
		}
		hero.Draw(batch)
		for i := range slimes {
			slimes[i].Draw(batch)
		}
		batch.Draw(win)

		// debug
		imd.Clear()
		imd.Color = colornames.Whitesmoke
		imd.Push(gunStart, gunEnd)
		imd.Line(2)

		imd.Color = colornames.Blueviolet
		//drawRect(imd, hero.Collider.Moved(origin))
		imd.Draw(win)

		// debug text
		win.SetMatrix(pixel.IM)
		mPosTxt.Draw(win, pixel.IM.Scaled(mPosTxt.Orig, 1))

		win.Update()
		engine.fpsHandler()

		dtDraw = time.Since(dtDrawSt).Seconds()
		if dtDraw > dtDrawMax {
			dtDrawMax = dtDraw
		}
	}
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")
var traceprofile = flag.String("traceprofile", "", "write trace profile to file")

func main() {
	if *memprofile != "" {
		runtime.MemProfileRate = 16
	}
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	if *traceprofile != "" {
		f, err := os.Create(*traceprofile)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		err = trace.Start(f)
		if err != nil {
			panic(err)
		}
		defer trace.Stop()

	}
	pixelgl.Run(run)
	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
		f.Close()
	}
}
