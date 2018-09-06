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

func TimeScheduler(init, rate float64) func(float64) float64 {
	next := init
	return func(t float64) float64 {
		next := next - t*rate
		return t + next
	}
}

var (
	engine *Engine
	world  *World
	hero   *Hero
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
		Bounds: pixel.R(0, 0, 1400, 800),
		VSync:  *vsync,
	}
	engine = NewEngine(&cfg)
	// engine.win.SetMonitor(pixelgl.PrimaryMonitor())

	trid := &pixel.TrianglesData{}
	batch := pixel.NewBatch(trid, tileset)

	tridBg := &pixel.TrianglesData{}
	batchBg := pixel.NewBatch(tridBg, tileset)

	imd := imdraw.New(nil)

	camera := NewCamera(engine.win)
	camera.Pos = pixel.V(216, 83)
	camera.Zoom = 4
	camera.Speed = 1

	// font
	atlas := text.NewAtlas(basicfont.Face7x13, text.ASCII)
	debugText := text.New(pixel.V(8, engine.win.Bounds().Max.Y-16), atlas)
	lostText := text.New(engine.win.Bounds().Center(), atlas)
	scoreText := text.New(engine.win.Bounds().Max.Add(pixel.V(-236, -36)), atlas)

	sprWall := pixel.NewSprite(tileset, frames[256-37])
	var sprBG []*pixel.Sprite
	sprBG = append(sprBG, pixel.NewSprite(tileset, frames[176]))
	sprBG = append(sprBG, pixel.NewSprite(tileset, frames[177]))
	sprBG = append(sprBG, pixel.NewSprite(tileset, frames[178]))
	sprBG = append(sprBG, pixel.NewSprite(tileset, frames[256-37]))
	sprBG = append(sprBG, pixel.NewSprite(tileset, frames[256-36]))
	world = NewWorld(40, 25, sSize, sprWall, sprBG, batchBg)

	spr := pixel.NewSprite(tileset, frames[1])
	hero = NewHero(
		spr,
		pixel.V(48, 100),
		90,
		400,
	)
	r := pixel.R(-spr.Frame().W()/2.5, -spr.Frame().H()/2.5, spr.Frame().W()/2.5, spr.Frame().H()/3)
	hero.Collider = &r

	sprBow := pixel.NewSprite(tileset, frames[28])
	bow := NewEntity(sprBow, pixel.ZV)
	bow.Color = colornames.Gold

	sprArrow := pixel.NewSprite(tileset, frames[26])
	sprStuckArrow := pixel.NewSprite(tileset, frames[27])
	arrows := make([]*Arrow, 3)
	for i := range arrows {
		arrows[i] = NewArrow(sprArrow, sprStuckArrow)
	}
	for i := range arrows {
		arrows[i].ToQuiver()
	}
	arrowsQ := make([]*Arrow, 0, 3)
	arrowsQ = append(arrowsQ, arrows...)
	var arrowInHand *Arrow
	timeToDrawArrow := 1.0
	drawArrowDone := engine.elapsed + timeToDrawArrow

	sprSlime := pixel.NewSprite(tileset, frames[15])
	slimes := make([]*Slime, 200)
	for i := range slimes {
		slimes[i] = NewSlime(sprSlime)
	}
	nextSlime := TimeScheduler(8.0, 0.01)
	slimeRecycle := 0

	targetFrameTime := 16500 * time.Microsecond
	gcOnFrame := 160
	gcFrame := 0
	// var gcTime time.Duration

	var (
		dtMax       float64
		dtUpdate    float64
		dtDraw      float64
		dtUpdateMax float64
		dtDrawMax   float64
	)

	nextSlimeTime := nextSlime(engine.elapsed)

	gameOver := false
	gameScore := 0

	win := engine.win
	// prewarm input
	for i := 0; i < 2; i++ {
		engine.fpsHandler()
		win.UpdateInput()
		win.Update()
	}

	for {
		for time.Since(engine.prevFrameStarted) < targetFrameTime {
			time.Sleep(10 * time.Microsecond)
		}
		engine.dt = time.Since(engine.prevFrameStarted).Seconds()
		engine.prevFrameStarted = time.Now()
		if engine.dt > dtMax {
			dtMax = engine.dt
		}

		gcFrame++
		if gcFrame >= gcOnFrame {
			gcFrame = 0
			dtMax = 0
			dtUpdateMax = 0
			dtDrawMax = 0
		}

		dtUpdateSt := time.Now()
		if win.JustPressed(pixelgl.KeyF) {
			if engine.win.Monitor() == nil {
				engine.win.SetMonitor(pixelgl.PrimaryMonitor())
			} else {
				engine.win.SetMonitor(nil)
			}
		}
		if win.Pressed(pixelgl.KeyEscape) {
			return
		}

		camera.Update()
		camMat := camera.GetMatrix()
		mousePos := camMat.Unproject(win.MousePosition())

		hero.Update()

		// bow
		if hero.Alive() {
			lookVec := mousePos.Sub(hero.Pos)
			dir := lookVec.Unit()
			bow.Pos = hero.Pos.Add(dir.Scaled(ArrowStartDistance - 3))
			bow.Angle = dir.Angle()

			lookDistance := pixel.Clamp(lookVec.Len(), 0, 64)
			lookAt := hero.Pos.Add(
				lookVec.Unit().Scaled(lookDistance))
			lookAt = lookAt.Add(hero.velocity.Scaled(0.64))
			camera.Follow(lookAt)
		}

		// arrows
		for _, a := range arrows {
			if a.Active {
				a.Update()
			}
		}

		if arrowInHand == nil && engine.elapsed > drawArrowDone {
			// Move arrow from the quiver to hands.
			if len(arrowsQ) > 0 {
				last := len(arrowsQ) - 1
				arrowInHand = arrowsQ[last]
				arrowsQ = arrowsQ[:last]
				arrowInHand.ToHands()
			}
		}

		if hero.Alive() {
			if win.JustPressed(pixelgl.MouseButton1) && arrowInHand != nil {
				arrowInHand.Fly(hero.Pos, mousePos, hero.velocity.Scaled(0.22))
				arrowInHand = nil
				if len(arrowsQ) > 0 {
					drawArrowDone = engine.elapsed + timeToDrawArrow
				}
			}

			heroColBig := hero.AbsCollider()
			quiverIdx := 0
			for i := range arrows {
				a := arrows[i]
				if a.State == ArrowStuck {
					if Collides(a.AbsCollider(), heroColBig) {
						if len(arrowsQ) == 0 {
							drawArrowDone = engine.elapsed + timeToDrawArrow
						}
						arrowsQ = append(arrowsQ, a)
						a.ToQuiver()
					}
				}

				if a.State == ArrowHands {
					a.AttachToHands(hero.Pos, mousePos)
				}
				if a.State == ArrowQuiver {
					a.AttachToQuiver(hero.Pos, quiverIdx)
					quiverIdx++
				}
			}
		}

		// slimes
		for i := range slimes {
			if slimes[i].Active {
				s := slimes[i]
				aliveBefore := s.Alive
				s.Update(arrows)
				if aliveBefore && !s.Alive {
					gameScore += int(math.Round(s.Pos.Sub(hero.Pos).Len() * (1 + engine.elapsed/1000)))
				}
			}
		}

		if engine.elapsed > nextSlimeTime {
			free := firstFreeSlime(slimes)
			if free == -1 {
				free = slimeRecycle
				slimeRecycle++
				if slimeRecycle >= len(slimes) {
					slimeRecycle = 0
				}
			}
			slimes[free].Spawn()
			nextSlimeTime = nextSlime(engine.elapsed)
		}

		if !gameOver && !hero.Alive() {
			gameOver = true
			lostText.Clear()
			fmt.Fprint(lostText, "Game Over!\nPress Esc to exit")
		}
		scoreText.Clear()
		fmt.Fprintf(scoreText, "Game score: %d", gameScore)

		// debug text
		debugText.Clear()
		// fmt.Fprintf(mPosTxt, "mpos: %6.3f %6.3f\n", mousePos.X, mousePos.Y)
		// fmt.Fprintf(mPosTxt, "hpos: %6.3f %6.3f\n", hero.Pos.X, hero.Pos.Y)
		// fmt.Fprintf(mPosTxt, "hvel: %6.3f %6.3f %6.3f\n", hero.velocity.X, hero.velocity.Y, hero.velocity.Len())
		// fmt.Fprintf(mPosTxt, "health: %6.1f\n", hero.health)
		fmt.Fprintf(debugText, "   fps: %3.0d\n", int(math.Round(1.0/engine.dt)))
		fmt.Fprintf(debugText, "cur dt: %6.5f\n", engine.dt)
		fmt.Fprintf(debugText, "max dt: %6.5f\n", dtMax)
		fmt.Fprintf(debugText, "dt upd: %6.5f\n", dtUpdateMax)
		fmt.Fprintf(debugText, "dt dra: %6.5f\n", dtDrawMax)

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
		for i := range slimes {
			if !slimes[i].Alive {
				slimes[i].Draw(batch)
			}
		}
		for _, a := range arrows {
			if a.State == ArrowStuck {
				a.Draw(batch)
			}
		}
		for i := range slimes {
			if slimes[i].Alive {
				slimes[i].Draw(batch)
			}
		}
		bow.Draw(batch)
		for _, a := range arrows {
			if a.State != ArrowStuck {
				a.Draw(batch)
			}
		}
		hero.Draw(batch)
		batch.Draw(win)

		imd.Clear()
		imd.Color = colornames.Blueviolet
		//drawRect(imd, hero.Collider.Moved(origin))
		imd.Draw(win)

		// debug text
		win.SetMatrix(pixel.IM)
		debugText.Draw(win, pixel.IM.Scaled(debugText.Orig, 1))
		if !hero.Alive() {
			lostText.DrawColorMask(win, pixel.IM.Scaled(lostText.Bounds().Center(), 6), colornames.Black)
			lostText.DrawColorMask(win, pixel.IM.Scaled(lostText.Bounds().Center(), 6).Moved(pixel.V(-4, 6)), colornames.White)
		}
		scoreText.Draw(win, pixel.IM.Scaled(scoreText.Orig, 2))

		engine.fpsHandler()
		win.Update()

		dtDraw = time.Since(dtDrawSt).Seconds()
		if dtDraw > dtDrawMax {
			dtDrawMax = dtDraw
		}
	}
}

var vsync = flag.Bool("vsync", false, "use vsync")

var (
	cpuprofile   = flag.String("cpuprofile", "", "write cpu profile to file")
	memprofile   = flag.String("memprofile", "", "write memory profile to `file`")
	traceprofile = flag.String("traceprofile", "", "write trace profile to file")
)

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
