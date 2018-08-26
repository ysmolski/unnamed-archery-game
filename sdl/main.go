package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime/pprof"
	"time"

	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

var width int32 = 800
var height int32 = 600

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

	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		log.Fatalf("Failed to init SDL: %v", err)
	}
	defer sdl.Quit()

	if err := ttf.Init(); err != nil {
		log.Fatalf("Failed to init TTF: %v", err)
	}
	defer ttf.Quit()

	window, err := sdl.CreateWindow("SDL from Go", sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_CENTERED, width, height, sdl.WINDOW_SHOWN)
	if err != nil {
		log.Fatalf("Failed to create window: %v", err)
	}
	defer window.Destroy()

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED|sdl.RENDERER_PRESENTVSYNC)
	if err != nil {
		log.Fatalf("Failed to get renderer: %v", err)
	}
	defer renderer.Destroy()
	if rInfo, err := renderer.GetInfo(); err == nil {
		log.Printf("Renderer: %v, Max texture: %v x %v", rInfo.Name, rInfo.RendererInfoData.MaxTextureWidth, rInfo.RendererInfoData.MaxTextureHeight)
	}

	if rw, rh, err := renderer.GetOutputSize(); err == nil {
		// Support for Retina displays. Override width and height values with
		// dimensions in pixels. These are dimensions where we can draw.
		log.Printf("Drawable dimensions: %v x %v", rw, rh)
		width = rw
		height = rh
	}

	font, err := ttf.OpenFont("lazy.ttf", 28)
	if err != nil {
		log.Fatalf("Cannot load font: %v", err)
	}

	tileset := TextureFromFile(renderer, "tileset.png")
	defer tileset.T.Destroy()
	tileset.T.SetColorMod(0xee, 0xee, 0xee)
	//tileset.T.SetColorMod(255, 255, 255)

	yellow := sdl.Color{0xff, 0xff, 0, 0xff}

	wSize := 32
	w := make([][]cell, wSize)
	for i := 0; i < wSize; i++ {
		w[i] = make([]cell, wSize)
		for j := 0; j < wSize; j++ {
			w[i][j].bg.R = uint8(rand.Intn(16))
			w[i][j].bg.G = uint8(rand.Intn(64) + 64)
			w[i][j].bg.A = 255
		}
	}

	for i := 0; i < 100; i++ {
		x := rand.Intn(32)
		y := rand.Intn(32)
		w[x][y].wall = NewWall(16*7, 16*5, 16, 16, tileset)
	}

	hero := &entity{xp: 50, yp: 50, maxV: 100}
	hero.Sprite.Set(&sdl.Rect{16 * 1, 16 * 0, 16, 16}, tileset)
	hero.collider = &sdl.Rect{hero.xp, hero.yp, 16, 16}

	wall := &sdl.Rect{128, 64, 32, 256}

	started := time.Now()
	var quit bool
	for !quit {
		dt := time.Since(started).Seconds()
		started = time.Now()
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch e := event.(type) {
			case *sdl.QuitEvent:
				quit = true
			case *sdl.KeyboardEvent:
				switch e.Keysym.Sym {
				case sdl.K_ESCAPE:
					quit = true
					continue
				}
				if e.Type == sdl.KEYDOWN && e.Repeat == 0 {
					switch e.Keysym.Sym {
					case sdl.K_w:
						hero.yv -= hero.maxV
					case sdl.K_s:
						hero.yv += hero.maxV
					case sdl.K_a:
						hero.xv -= hero.maxV
					case sdl.K_d:
						hero.xv += hero.maxV
					}
				}
				if e.Type == sdl.KEYUP && e.Repeat == 0 {
					switch e.Keysym.Sym {
					case sdl.K_w:
						hero.yv += hero.maxV
					case sdl.K_s:
						hero.yv -= hero.maxV
					case sdl.K_a:
						hero.xv += hero.maxV
					case sdl.K_d:
						hero.xv -= hero.maxV
					}
				}
				// fmt.Printf("%v\n", e)
				// fmt.Println(hero)
			}
		}
		// update the world
		hero.move(dt, wall)

		// draw the bg
		renderer.SetDrawColor(128, 128, 128, 0xFF)
		renderer.Clear()

		// draw grass
		for x := int32(0); int(x) < len(w); x++ {
			for y := int32(0); int(y) < len(w[x]); y++ {
				c := w[x][y]
				renderer.SetDrawColor(c.bg.R, c.bg.G, c.bg.B, c.bg.A)
				renderer.FillRect(&sdl.Rect{x * 16, y * 16, 16, 16})
				//tileset.Render(x*16, y*16, &sdl.Rect{(x % 16) * 16, (y % 16) * 16, 16, 16})
			}
		}

		for i := 0; i < wSize; i++ {
			for j := 0; j < wSize; j++ {
				c := w[i][j]
				if c.wall != nil {
					c.wall.Sprite.Render(int32(16*i), int32(16*j))
				}
			}
		}
		hero.render()
		renderer.DrawRect(wall)

		// fps
		fpsText := TextureFromText(renderer, font, fmt.Sprint(int(dt*1000)), yellow)
		fpsText.Render(width-50, height-35, nil)

		renderer.SetDrawColor(0xff, 0xff, 0xff, 0xFF)
		renderer.DrawLine(width-10, height, width-10, height-int32(dt*1000))
		renderer.Present()
	}
}

type Sprite struct {
	clip    *sdl.Rect
	texture *Texture
}

func (s *Sprite) Set(clip *sdl.Rect, tex *Texture) {
	s.clip = clip
	s.texture = tex
}

func (s *Sprite) Render(x, y int32) {
	s.texture.Render(x, y, s.clip)
}

type entity struct {
	Sprite
	xp, yp   int32 // position in the world
	xv, yv   int32 // velocity per axis
	maxV     int32
	collider *sdl.Rect
}

func (e *entity) render() {
	xr := (e.xp + 8) / 16 * 16
	yr := (e.yp + 8) / 16 * 16
	e.Sprite.texture.R.SetDrawColor(120, 120, 120, 255)
	e.Sprite.texture.R.DrawRect(&sdl.Rect{xr, yr, 16, 16})
	e.Sprite.Render(e.xp, e.yp)
}

func (e *entity) move(dt float64, wall *sdl.Rect) {
	xdelta := int32(float64(e.xv) * dt)
	e.xp += xdelta
	e.collider.X = e.xp
	if e.xp < 0 || e.xp+16 > width || collides(e.collider, wall) {
		e.xp -= xdelta
		e.collider.X = e.xp
	}
	ydelta := int32(float64(e.yv) * dt)
	e.yp += ydelta
	e.collider.Y = e.yp
	if e.yp < 0 || e.yp+16 > height || collides(e.collider, wall) {
		e.yp -= ydelta
		e.collider.Y = e.yp
	}
}

type cell struct {
	bg   sdl.Color
	wall *Wall
}

type Object interface {
	Passable() bool
}

func NewWall(x, y, w, h int32, tex *Texture) *Wall {
	return &Wall{Sprite{&sdl.Rect{x, y, w, h}, tex}}
}

type Wall struct {
	Sprite
}

func (w *Wall) Passable() bool {
	return false
}

type Texture struct {
	Width, Height int32
	T             *sdl.Texture
	R             *sdl.Renderer
}

func collides(a, b *sdl.Rect) bool {
	aLeft := a.X
	aRight := a.X + a.W
	aTop := a.Y
	aBottom := a.Y + a.H
	bLeft := b.X
	bRight := b.X + b.W
	bTop := b.Y
	bBottom := b.Y + b.H
	if aBottom <= bTop || bBottom <= aTop {
		return false
	}
	if aRight <= bLeft || bRight <= aLeft {
		return false
	}
	return true
}

func TextureFromFile(r *sdl.Renderer, filename string) *Texture {
	s, err := img.Load(filename)
	if err != nil {
		log.Fatalf("Cannot load image: %v", err)
	}
	//	s.SetColorKey(true, sdl.MapRGB(s.Format, 0, 0xFF, 0xFF)) // BG is cyan.
	t, err := r.CreateTextureFromSurface(s)
	if err != nil {
		log.Fatalf("Cannot create texture: %v", err)
	}
	te := &Texture{Width: s.W, Height: s.H, T: t, R: r}
	return te
}

func TextureFromText(r *sdl.Renderer, font *ttf.Font, text string, color sdl.Color) *Texture {
	s, err := font.RenderUTF8Solid(text, color)
	if err != nil {
		log.Fatalf("Cannot render font: %v", err)
	}
	t, err := r.CreateTextureFromSurface(s)
	if err != nil {
		log.Fatalf("Cannot create texture: %v", err)
	}
	te := &Texture{Width: s.W, Height: s.H, T: t, R: r}
	return te
}

func (t *Texture) Render(x, y int32, clip *sdl.Rect) {
	destQuad := &sdl.Rect{X: x, Y: y, W: t.Width, H: t.Height}
	if clip != nil {
		destQuad.W = clip.W
		destQuad.H = clip.H
	}
	t.R.Copy(t.T, clip, destQuad)
}
