package qoi

import (
	"image"
	"image/color"
)

type Colorspace uint8

const (
	SRGB   Colorspace = 0
	Linear Colorspace = 1
)

type Image struct {
	Pix        []byte
	Width      int
	Height     int
	Channels   uint8
	Colorspace Colorspace
}

func (img *Image) ColorModel() color.Model {
	return color.NRGBAModel
}

func (img *Image) Bounds() image.Rectangle {
	return image.Rect(0, 0, img.Width, img.Height)
}

func (img *Image) At(x, y int) color.Color {
	if img.Channels == 4 {
		return color.NRGBA{R: img.Pix[(y*img.Width+x)*int(img.Channels)], G: img.Pix[(y*img.Width+x)*int(img.Channels)+1], B: img.Pix[(y*img.Width+x)*int(img.Channels)+2], A: img.Pix[(y*img.Width+x)*int(img.Channels)+3]}
	}
	return color.NRGBA{R: img.Pix[(y*img.Width+x)*int(img.Channels)], G: img.Pix[(y*img.Width+x)*int(img.Channels)+1], B: img.Pix[(y*img.Width+x)*int(img.Channels)+2], A: 255}
}
