package util

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/nfnt/resize"
)

// ResizeImage resizes an image to the specified dimensions using the provided interpolation function.
// nolint:unparam
func ResizeImage(
	img image.Image,
	width int,
	height int,
	interpolationFunction resize.InterpolationFunction,
) (image.Image, error) {
	// nolint: gosec
	resizedImage := resize.Resize(uint(width), uint(height), img, interpolationFunction)

	return resizedImage, nil
}

func RemoveImageAlphaChannel(img image.Image) image.Image {
	// Create a new RGBA image with the same size as the original image
	rgba := image.NewRGBA(img.Bounds())

	// Draw the original image onto the new RGBA image
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)

	// Remove the alpha channel by setting alpha to 255 (fully opaque) for all pixels
	for y := rgba.Bounds().Min.Y; y < rgba.Bounds().Max.Y; y++ {
		for x := rgba.Bounds().Min.X; x < rgba.Bounds().Max.X; x++ {
			rgba.SetRGBA(x, y, color.RGBA{
				R: rgba.RGBAAt(x, y).R,
				G: rgba.RGBAAt(x, y).G,
				B: rgba.RGBAAt(x, y).B,
				A: 255, // Set alpha to fully opaque
			})
		}
	}

	return rgba
}
