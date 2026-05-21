package main

import (
	"image"
	"image/color"
)

// makeRedImage returns a w×h solid red *image.RGBA, used by the
// run_test.go logo-overlay test fixture.
func makeRedImage(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	red := color.RGBA{R: 255, A: 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, red)
		}
	}
	return img
}
