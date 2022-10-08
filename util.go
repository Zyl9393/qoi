package qoi

import "image"

func isOpaqueImage(im image.Image) bool {
	// Check if image has Opaque() method:
	if oim, ok := im.(interface {
		Opaque() bool
	}); ok {
		return oim.Opaque() // It does, call it and return its result!
	}

	// loop through all pixels and check manually:
	rect := im.Bounds()
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			if _, _, _, a := im.At(x, y).RGBA(); a != 0xffff {
				return false // Found a non-opaque pixel: image is non-opaque
			}
		}
	}

	return true // All pixels are opaque, so is the image
}
