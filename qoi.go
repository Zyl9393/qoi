package qoi

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
)

func init() {
	image.RegisterFormat("qoi", qoiMagic, decode, DecodeConfig)
}

type Header struct {
	magic      [4]byte
	width      uint32
	height     uint32
	channels   uint8
	colorspace Colorspace
}

const (
	qoi_INDEX byte = 0b00_000000
	qoi_DIFF  byte = 0b01_000000
	qoi_LUMA  byte = 0b10_000000
	qoi_RUN   byte = 0b11_000000
	qoi_RGB   byte = 0b1111_1110
	qoi_RGBA  byte = 0b1111_1111

	qoi_MASK_2 byte = 0b11_000000
)

var qoiEnd = []byte{0, 0, 0, 0, 0, 0, 0, 0b00000001}

const qoiMagic = "qoif"

const qoiPixelsMax = 400_000_000 // 400 million pixels ought to be enough for anybody

func qoi_COLOR_HASH(r, g, b, a byte) byte {
	return byte(r*3 + g*5 + b*7 + a*11)
}

type pixel [4]byte

func DecodeConfig(reader io.Reader) (cfg image.Config, err error) {
	header, err := DecodeHeader(reader)
	if err != nil {
		return cfg, err
	}
	return image.Config{ColorModel: color.NRGBAModel, Width: int(header.width), Height: int(header.height)}, nil
}

func decode(r io.Reader) (image.Image, error) {
	return Decode(r)
}

func Decode(reader io.Reader) (*Image, error) {
	header, err := DecodeHeader(reader)
	if err != nil {
		return nil, err
	}
	pix := make([]uint8, header.width*header.height*uint32(header.channels))
	img := &Image{
		Pix:        pix,
		Width:      int(header.width),
		Height:     int(header.height),
		Channels:   header.channels,
		Colorspace: header.colorspace,
	}
	return img, decodeBody(reader, pix, int(img.Channels), img.Width*int(img.Channels))
}

func decodeBody(r io.Reader, dest []uint8, bytesPerPixel int, stride int) (err error) {
	in := bufio.NewReaderSize(r, 250)
	numPixels := len(dest) / bytesPerPixel

	var b1, b2 byte

	var index [64]pixel

	run := 0
	numDecodedPixels := 0

	px := pixel{0, 0, 0, 255}
	for numDecodedPixels < numPixels {
		if run > 0 {
			run--
		} else {
			b1, err = in.ReadByte()
			if err == io.EOF {
				return fmt.Errorf("unexpected EOF after %d pixels: expected %d", numDecodedPixels, numPixels)
			}
			if err != nil {
				return err
			}

			switch {
			case b1 == qoi_RGB:
				_, err = io.ReadFull(in, px[:3])
				if err != nil {
					return err
				}
			case b1 == qoi_RGBA:
				_, err = io.ReadFull(in, px[:])
				if err != nil {
					return err
				}
			case b1&qoi_MASK_2 == qoi_INDEX:
				px = index[b1]
			case b1&qoi_MASK_2 == qoi_DIFF:
				px[0] += ((b1 >> 4) & 0x03) - 2
				px[1] += ((b1 >> 2) & 0x03) - 2
				px[2] += (b1 & 0x03) - 2
			case b1&qoi_MASK_2 == qoi_LUMA:
				b2, err = in.ReadByte()
				if err != nil {
					return err
				}
				vg := (b1 & 0b00111111) - 32
				px[0] += vg - 8 + ((b2 >> 4) & 0x0f)
				px[1] += vg
				px[2] += vg - 8 + (b2 & 0x0f)
			case b1&qoi_MASK_2 == qoi_RUN:
				run = int(b1 & 0b00111111)
			default:
				px = pixel{255, 0, 255, 255} // should not happen
			}

			index[int(qoi_COLOR_HASH(px[0], px[1], px[2], px[3]))&0b111111] = px
		}

		copy(dest[:bytesPerPixel], px[:bytesPerPixel])
		dest = dest[bytesPerPixel:]
		numDecodedPixels++
	}
	return nil
}

// Decode decodes QOI image data from r into dest, until all pixels are written.
// If dest cannot fit the image, an error is returned.
func DecodeIntoBuffer(r io.Reader, dest []byte) (*Image, error) {
	header, err := DecodeHeader(r)
	if err != nil {
		return nil, fmt.Errorf("could not decode header: %w", err)
	}
	numPixels := int(header.width * header.height)
	if numPixels == 0 {
		return nil, nil
	}
	bytesPerPixel := int(header.channels)
	if numPixels*bytesPerPixel > len(dest) {
		return nil, fmt.Errorf("dest of size %d bytes cannot fit image data totalling %d bytes", len(dest), numPixels*bytesPerPixel)
	}
	img := &Image{
		Pix:        dest[:numPixels*bytesPerPixel],
		Width:      int(header.width),
		Height:     int(header.height),
		Channels:   header.channels,
		Colorspace: header.colorspace,
	}
	return img, decodeBody(r, img.Pix, int(img.Channels), img.Width*int(img.Channels))
}

// Encode encodes img as a QOI file and writes it to w.
func Encode(w io.Writer, img image.Image) error {
	out := bufio.NewWriter(w)

	minX := img.Bounds().Min.X
	maxX := img.Bounds().Max.X
	minY := img.Bounds().Min.Y
	maxY := img.Bounds().Max.Y
	width := maxX - minX
	height := maxY - minY

	numPixels := width * height
	if numPixels == 0 {
		return errors.New("bad image size 0")
	} else if numPixels >= qoiPixelsMax {
		return fmt.Errorf("image must have less than %d pixels total", qoiPixelsMax)
	}
	bytesPerPixel := 3
	if !isOpaqueImage(img) {
		bytesPerPixel++
	}

	// write header to output
	if err := binary.Write(out, binary.BigEndian, []byte(qoiMagic)); err != nil {
		return err
	}
	// width
	if err := binary.Write(out, binary.BigEndian, uint32(width)); err != nil {
		return err
	}
	// height
	if err := binary.Write(out, binary.BigEndian, uint32(height)); err != nil {
		return err
	}
	// channels
	if err := binary.Write(out, binary.BigEndian, uint8(bytesPerPixel)); err != nil {
		return err
	}
	// sRGB with linear alpha
	if err := binary.Write(out, binary.BigEndian, uint8(0)); err != nil {
		return err
	}

	var index [64]pixel
	px_prev := pixel{0, 0, 0, 255}
	run := 0

	widthMinusOne := width - 1
	heightMinusOne := height - 1
	var px pixel

	for y := minY; y < maxY; y++ {
		for x := minX; x < maxX; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			px = pixel{c.R, c.G, c.B, c.A}

			if px == px_prev {
				run++
				last_pixel := x == widthMinusOne && y == heightMinusOne
				if run == 62 || last_pixel {
					out.WriteByte(qoi_RUN | byte(run-1))
					run = 0
				}
			} else {
				if run > 0 {
					out.WriteByte(qoi_RUN | byte(run-1))
					run = 0
				}
				var index_pos byte = qoi_COLOR_HASH(px[0], px[1], px[2], px[3]) & 0b111111
				if index[index_pos] == px {
					out.WriteByte(qoi_INDEX | index_pos)
				} else {
					index[index_pos] = px

					if px[3] == px_prev[3] {
						vr := int8(int(px[0]) - int(px_prev[0]))
						vg := int8(int(px[1]) - int(px_prev[1]))
						vb := int8(int(px[2]) - int(px_prev[2]))

						vg_r := vr - vg
						vg_b := vb - vg

						if vr > -3 && vr < 2 && vg > -3 && vg < 2 && vb > -3 && vb < 2 {
							out.WriteByte(qoi_DIFF | byte((vr+2)<<4|(vg+2)<<2|(vb+2)))
						} else if vg_r > -9 && vg_r < 8 && vg > -33 && vg < 32 && vg_b > -9 && vg_b < 8 {
							out.WriteByte(qoi_LUMA | byte(vg+32))
							out.WriteByte(byte((vg_r+8)<<4) | byte(vg_b+8))
						} else {
							out.WriteByte(qoi_RGB)
							out.WriteByte(px[0])
							out.WriteByte(px[1])
							out.WriteByte(px[2])
						}

					} else {
						out.WriteByte(qoi_RGBA)
						for i := 0; i < 4; i++ {
							out.WriteByte(px[i])
						}
					}

				}
			}

			px_prev = px
		}
	}
	binary.Write(out, binary.BigEndian, uint32(0)) // padding
	binary.Write(out, binary.BigEndian, uint32(1)) // padding

	return out.Flush()
}

// DecodeHeader decodes only the header from the beginning of a QOI image and returns it, if it is valid.
func DecodeHeader(r io.Reader) (header Header, err error) {
	err = binary.Read(r, binary.BigEndian, &header.magic)
	if err != nil {
		return Header{}, fmt.Errorf("could not read header magic: %w", err)
	}
	err = binary.Read(r, binary.BigEndian, &header.width)
	if err != nil {
		return Header{}, fmt.Errorf("could not read width: %w", err)
	}
	err = binary.Read(r, binary.BigEndian, &header.height)
	if err != nil {
		return Header{}, fmt.Errorf("could not read height: %w", err)
	}
	err = binary.Read(r, binary.BigEndian, &header.channels)
	if err != nil {
		return Header{}, fmt.Errorf("could not read channels: %w", err)
	}
	err = binary.Read(r, binary.BigEndian, &header.colorspace)
	if err != nil {
		return Header{}, fmt.Errorf("could not read colorspace: %w", err)
	}
	if string(header.magic[:4]) != qoiMagic {
		return Header{}, fmt.Errorf("bad magic")
	}
	if header.channels < 3 || header.channels > 4 {
		return Header{}, fmt.Errorf("invalid amount of channels %d: must be 3 or 4", header.channels)
	}
	if header.colorspace != SRGB && header.colorspace != Linear {
		return Header{}, fmt.Errorf("invalid colorspace %d: must be 0 (sRGB) or 1 (linear RGB)", header.colorspace)
	}
	return header, nil
}
