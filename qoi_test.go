package qoi_test

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"testing"

	"github.com/Zyl9393/qoi"
	testdataloader "github.com/peteole/testdata-loader"
)

func TestDecode(t *testing.T) {
	pngContent := testdataloader.GetTestFile("testdata/cyberpanel1.png")
	img, err := png.Decode(bytes.NewReader(pngContent))
	if err != nil {
		t.Fatal(err)
	}
	qoiEncode := bytes.NewBuffer(nil)
	err = qoi.Encode(qoiEncode, img)
	if err != nil {
		t.Fatal(err)
	}
	decodeImg, _, err := image.Decode(qoiEncode)
	if err != nil {
		t.Fatal(err)
	}
	err = imageEquals(decodeImg, img)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDecodeWithBuffer(t *testing.T) {
	pngContent := testdataloader.GetTestFile("testdata/cyberpanel1.png")
	img, err := png.Decode(bytes.NewReader(pngContent))
	if err != nil {
		t.Fatal(err)
	}
	qoiEncode := bytes.NewBuffer(nil)
	err = qoi.Encode(qoiEncode, img)
	if err != nil {
		t.Fatal(err)
	}
	bigBuf := make([]byte, 1024*1024*4)
	decodeImg, err := qoi.DecodeIntoBuffer(qoiEncode, bigBuf)
	if err != nil {
		t.Fatal(err)
	}
	err = imageEquals(decodeImg, img)
	if err != nil {
		t.Fatal(err)
	}
}

func imageEquals(a, b image.Image) error {
	if !sameRectDimensions(a.Bounds(), b.Bounds()) {
		return fmt.Errorf("dimensions not equal")
	}
	ar := a.Bounds()
	br := b.Bounds()
	aMinX, aMinY := ar.Min.X, ar.Min.Y
	bMinX, bMinY := br.Min.X, br.Min.Y
	width := ar.Dx()
	height := ar.Dy()
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if a.At(aMinX+x, aMinY+y) != b.At(bMinX+x, bMinY+y) {
				return fmt.Errorf("images not equal")
			}
		}
	}
	return nil
}

func sameRectDimensions(a, b image.Rectangle) bool {
	return a.Dx() == b.Dx() && a.Dy() == b.Dy()
}
