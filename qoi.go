package qoi

/*

QOI - The “Quite OK Image” format for fast, lossless image compression

Original version by Dominic Szablewski - https://phoboslab.org
Go version by Makapuf makapuf2@gmail.com

-- LICENSE: The MIT License(MIT)

Copyright(c) 2021 Dominic Szablewski

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files(the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and / or sell copies
of the Software, and to permit persons to whom the Software is furnished to do
so, subject to the following conditions :
The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.


*/

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io"
)

const (
	QOI_INDEX   byte = 0b00_000000
	QOI_RUN_8   byte = 0b010_00000
	QOI_RUN_16  byte = 0b011_00000
	QOI_DIFF_8  byte = 0b10_000000
	QOI_DIFF_16 byte = 0b110_00000
	QOI_DIFF_24 byte = 0b1110_0000
	QOI_COLOR   byte = 0b1111_0000

	QOI_MASK_2 byte = 0b11_000000
	QOI_MASK_3 byte = 0b111_00000
	QOI_MASK_4 byte = 0b1111_0000
)

const QOIMagic = "qoif"

func QOI_COLOR_HASH(r, g, b, a byte) byte {
	return byte(r ^ g ^ b ^ a)
}

func Decode(r io.Reader) (image.Image, error) {
	// read header
	var (
		header [4]byte
		width  uint16
		height uint16
		size   uint32
	)

	b := bufio.NewReader(r)

	err := binary.Read(b, binary.LittleEndian, &header)
	if err != nil {
		return nil, err
	}
	if string(header[:]) != QOIMagic {
		return nil, fmt.Errorf("bad header")
	}

	binary.Read(b, binary.LittleEndian, &width)
	binary.Read(b, binary.LittleEndian, &height)
	binary.Read(b, binary.LittleEndian, &size)

	img := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))

	var index [64][4]byte

	run := 0

	pixels := img.Pix // pixels yet to write
	px := [4]byte{0, 0, 0, 255}
	for len(pixels) > 0 {
		if run > 0 {
			run--
		} else {

			b1, err := b.ReadByte()
			// read and handle end of file
			if err == io.EOF {
				return img, nil
			}
			if err != nil {
				return nil, err
			}
			switch {
			case (b1 & QOI_MASK_2) == QOI_INDEX:
				px = index[b1^QOI_INDEX]

			case (b1 & QOI_MASK_3) == QOI_RUN_8:
				run = int(b1 & 0x1f)

			case ((b1 & QOI_MASK_3) == QOI_RUN_16):
				b2, err := b.ReadByte()
				if err != nil {
					return nil, err
				}
				run = int((((b1 & 0x1f) << 8) | (b2)) + 32)

			case ((b1 & QOI_MASK_2) == QOI_DIFF_8):
				px[0] += ((b1 >> 4) & 0x03) - 1
				px[1] += ((b1 >> 2) & 0x03) - 1
				px[2] += (b1 & 0x03) - 1

			case ((b1 & QOI_MASK_3) == QOI_DIFF_16):
				b2, err := b.ReadByte()
				if err != nil {
					return nil, err
				}
				px[0] += (b1 & 0x1f) - 15
				px[1] += (b2 >> 4) - 7
				px[2] += (b2 & 0x0f) - 7

			case ((b1 & QOI_MASK_4) == QOI_DIFF_24):
				b2, err := b.ReadByte()
				if err != nil {
					return nil, err
				}
				b3, err := b.ReadByte()
				if err != nil {
					return nil, err
				}

				px[0] += (((b1 & 0x0f) << 1) | (b2 >> 7)) - 15
				px[1] += ((b2 & 0x7c) >> 2) - 15
				px[2] += (((b2 & 0x03) << 3) | ((b3 & 0xe0) >> 5)) - 15
				px[3] += (b3 & 0x1f) - 15

			case (b1 & QOI_MASK_4) == QOI_COLOR:
				if b1&8 != 0 {
					b2, err := b.ReadByte()
					if err != nil {
						return nil, err
					}
					px[0] = b2
				}
				if b1&4 != 0 {
					b2, err := b.ReadByte()
					if err != nil {
						return nil, err
					}
					px[1] = b2
				}
				if b1&2 != 0 {
					b2, err := b.ReadByte()
					if err != nil {
						return nil, err
					}
					px[2] = b2
				}
				if b1&1 != 0 {
					b2, err := b.ReadByte()
					if err != nil {
						return nil, err
					}
					px[3] = b2
				}
			default:
				px = [4]byte{255, 0, 255, 255}
			}

			index[int(QOI_COLOR_HASH(px[0], px[1], px[2], px[3]))%len(index)] = px
		}

		// TODO stride ..
		copy(pixels[:4], px[:])
		pixels = pixels[4:] // advance
	}
	return img, nil
}

func Encode(w io.Writer, m image.Image) error {
	minX := m.Bounds().Min.X
	maxX := m.Bounds().Max.X
	minY := m.Bounds().Min.Y
	maxY := m.Bounds().Max.Y

	// output buffer. We can't use w since it would need to be a bytewriter AND seeker to write the size
	// set its capacity to maxsize/2
	var out bytes.Buffer
	out.Grow((maxX - minX) * (maxY - minY) / 2)

	// write header to output
	err := binary.Write(&out, binary.LittleEndian, []byte(QOIMagic))
	err = binary.Write(&out, binary.LittleEndian, uint16(maxX-minX)) // width
	err = binary.Write(&out, binary.LittleEndian, uint16(maxY-minY)) // height
	size_pos := out.Len()
	err = binary.Write(&out, binary.LittleEndian, uint32(0)) // size, will be fixed later
	_ = err

	// TODO use a RGBA image / pix directly (faster)

	var index [64]color.RGBA
	px_prev := color.RGBA{0, 0, 0, 255}
	run := 0

	for y := minY; y < maxY; y++ {
		for x := minX; x < maxX; x++ {
			px := m.At(x, y).(color.RGBA)
			if px == px_prev {
				run++
			}

			last_pixel := x == (maxX-1) && y == (maxY-1)
			if run > 0 && (run == 0x2020 || px != px_prev || last_pixel) {
				if run < 33 {
					out.WriteByte(QOI_RUN_8 | byte(run-1))
				} else {
					run -= 33
					out.WriteByte(QOI_RUN_16 | byte(run>>8))
					out.WriteByte(byte(run & 0xff))
				}
				run = 0
			}

			if px != px_prev {

				px_r, px_g, px_b, px_a := px.RGBA()

				var index_pos byte = QOI_COLOR_HASH(byte(px_r>>8), byte(px_g>>8), byte(px_b>>8), byte(px_a>>8)) % 64
				if index[index_pos] == px {
					out.WriteByte(QOI_INDEX | index_pos)
				} else {
					index[index_pos] = px

					px_prev_r, px_prev_g, px_prev_b, px_prev_a := px_prev.RGBA()
					vr := (int(px_r) - int(px_prev_r)) >> 8
					vg := (int(px_g) - int(px_prev_g)) >> 8
					vb := (int(px_b) - int(px_prev_b)) >> 8
					va := (int(px_a) - int(px_prev_a)) >> 8

					if vr > -16 && vr < 17 && vg > -16 && vg < 17 && vb > -16 && vb < 17 && va > -16 && va < 17 {
						switch {
						case va == 0 && vr > -2 && vr < 3 && vg > -2 && vg < 3 && vb > -2 && vb < 3:
							out.WriteByte(QOI_DIFF_8 | byte(((vr+1)<<4)|(vg+1)<<2|(vb+1)))
						case va == 0 && vr > -16 && vr < 17 && vg > -8 && vg < 9 && vb > -8 && vb < 9:
							out.WriteByte(QOI_DIFF_16 | byte(vr+15))
							out.WriteByte(byte(((vg + 7) << 4) | (vb + 7)))
						default:
							out.WriteByte(QOI_DIFF_24 | byte((vr+15)>>1))
							out.WriteByte(byte(((vr + 15) << 7) | ((vg + 15) << 2) | ((vb + 15) >> 3)))
							out.WriteByte(byte(((vb + 15) << 5) | (va + 15)))
						}
					} else {
						mask := QOI_COLOR
						if vr != 0 {
							mask |= 1 << 3
						}
						if vg != 0 {
							mask |= 1 << 2
						}
						if vb != 0 {
							mask |= 1 << 1
						}
						if va != 0 {
							mask |= 1 << 0
						}
						out.WriteByte(mask)
						if vr != 0 {
							out.WriteByte(byte(px_r >> 8))
						}
						if vg != 0 {
							out.WriteByte(byte(px_g >> 8))
						}
						if vb != 0 {
							out.WriteByte(byte(px_b >> 8))
						}
						if va != 0 {
							out.WriteByte(byte(px_a >> 8))
						}
					}

				}
			}

			px_prev = px
		}
	}
	binary.Write(&out, binary.LittleEndian, uint32(0))                                 // padding
	binary.LittleEndian.PutUint16(out.Bytes()[size_pos:], uint16(len(out.Bytes())-12)) // fix size

	w.Write(out.Bytes())

	return nil
}

// unimplemented ...
func DecodeConfig(_ io.Reader) (image.Config, error) {
	return image.Config{}, fmt.Errorf("not implemented yet.")
}

func init() {
	image.RegisterFormat("qoi", QOIMagic, Decode, DecodeConfig)
}