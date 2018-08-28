package main

import "github.com/faiface/pixel"

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
