package main

import (
	"image/color"
	"math/rand"

	"github.com/faiface/pixel"
	"golang.org/x/image/colornames"
)

type CellType uint8

const (
	CellEmpty CellType = iota
	CellWall
	CellStone
	numberOfCellTypes
)

type World struct {
	gridSize      int // the side of one grid element
	width, height int
	cells         [][]CellType

	staticBatch *pixel.Batch
	wallMats    []pixel.Matrix
	wallColors  []color.RGBA
	wallSprite  *pixel.Sprite

	floorSpriteIdx []int
	floorMats      []pixel.Matrix
	floorColors    []color.RGBA
	floorSprites   []*pixel.Sprite
}

var wallAltColors = [...]color.RGBA{
	color.RGBA{0x95, 0x8F, 0x82, 0xFF},
	color.RGBA{0xA8, 0x9B, 0x94, 0xFF},
	color.RGBA{0x9E, 0x99, 0x8D, 0xFF},
}

var floorColors = [...]color.RGBA{
	color.RGBA{0, 21, 36, 255},
	color.RGBA{0, 24, 38, 255},
	color.RGBA{1, 26, 41, 255},
	color.RGBA{1, 32, 43, 255},
	color.RGBA{1, 36, 46, 255},
	color.RGBA{0, 38, 49, 255},
}

func NewWorld(width, height, gridSize int, sprWall *pixel.Sprite, sprFloor []*pixel.Sprite, batch *pixel.Batch) *World {
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

	// Generate wall tiles.
	w.staticBatch = batch
	w.staticBatch.Clear()
	w.wallSprite = sprWall
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			if w.cells[x][y] == CellWall {
				n := rand.Intn(8)
				mat := pixel.IM
				col := colornames.Gray
				if n == 0 {
					mat = mat.Rotated(pixel.ZV, (rand.Float64()-0.5)/5)
					mat = mat.Moved(pixel.V(
						float64(rand.Intn(3)-1),
						float64(rand.Intn(3)-1),
					))
				}
				mat = mat.Moved(pixel.V(float64(x*w.gridSize), float64(y*w.gridSize)))
				if n < 3 {
					ci := rand.Intn(len(wallAltColors))
					col = wallAltColors[ci]
				}
				w.wallMats = append(w.wallMats, mat)
				w.wallColors = append(w.wallColors, col)
			}
		}
	}

	// Generate floor tiles.
	w.floorSprites = append(w.floorSprites, sprFloor...)
	for x := 1; x < width-1; x++ {
		for y := 1; y < height-1; y++ {
			n := rand.Intn(100)
			if n > 8 {
				continue
			}
			v := pixel.V(float64(x*w.gridSize), float64(y*w.gridSize))

			mat := pixel.IM
			if n < 10 {
				mat = mat.Rotated(pixel.ZV, (rand.Float64()-0.5)/5)
				mat = mat.Moved(pixel.V(
					float64(rand.Intn(3)-1),
					float64(rand.Intn(3)-1),
				))
			}
			mat = mat.Moved(v)

			col := floorColors[rand.Intn(len(floorColors))]
			w.floorSpriteIdx = append(w.floorSpriteIdx, rand.Intn(len(w.floorSprites)))
			w.floorMats = append(w.floorMats, mat)
			w.floorColors = append(w.floorColors, col)
		}
	}

	// Write static background to staticBatch.
	for i := range w.floorMats {
		w.floorSprites[w.floorSpriteIdx[i]].DrawColorMask(w.staticBatch, w.floorMats[i], w.floorColors[i])
	}
	for i := range w.wallMats {
		w.wallSprite.DrawColorMask(w.staticBatch, w.wallMats[i], w.wallColors[i])
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

func (w *World) RandomVec() pixel.Vec {
	return pixel.V(
		rand.Float64()*float64((w.width-3)*w.gridSize)+float64(w.gridSize),
		rand.Float64()*float64((w.height-3)*w.gridSize)+float64(w.gridSize),
	)
}

func (w *World) GetColliders(collider pixel.Rect) []pixel.Rect {
	x1 := w.spaceToGrid(collider.Min.X)
	y1 := w.spaceToGrid(collider.Min.Y)
	x2 := w.spaceToGrid(collider.Max.X) + 1
	y2 := w.spaceToGrid(collider.Max.Y) + 1

	var r []pixel.Rect
	halfSize := float64(w.gridSize / 2)
	for x := x1; x <= x2; x++ {
		for y := y1; y <= y2; y++ {
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

func (w *World) Draw(t pixel.Target) {
	w.staticBatch.Draw(t)
	// for i := range w.floorMats {
	// 	w.floorSprites[w.floorSpriteIdx[i]].DrawColorMask(t, w.floorMats[i], w.floorColors[i])
	// }
	// for i := range w.wallMats {
	// 	w.wallSprite.DrawColorMask(t, w.wallMats[i], w.wallColors[i])
	// }
}
