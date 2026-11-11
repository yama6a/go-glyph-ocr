package glyphocr

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testThreshold = 200

func TestNewInkBitmapImageTypes(t *testing.T) {
	t.Parallel()
	nrgba := image.NewNRGBA(image.Rect(0, 0, 6, 1))
	for i, c := range []color.NRGBA{
		{255, 255, 255, 0},
		{255, 255, 255, 0x42},
		{255, 255, 255, 0x43},
		{255, 255, 255, 255},
		{0, 0, 0, 255},
		{255, 255, 0, 255},
	} {
		nrgba.SetNRGBA(i, 0, c)
	}

	nrgba64 := image.NewNRGBA64(image.Rect(0, 0, 4, 1))
	for i, a := range []uint16{0, 0x4200, 0x4300, 0xFFFF} {
		nrgba64.SetNRGBA64(i, 0, color.NRGBA64{0xFFFF, 0xFFFF, 0xFFFF, a})
	}

	palNRGBA := image.NewPaletted(image.Rect(0, 0, 4, 1), color.Palette{
		color.NRGBA{255, 255, 255, 0},
		color.NRGBA{255, 255, 255, 66},
		color.NRGBA{255, 255, 255, 67},
		color.NRGBA{255, 255, 255, 255},
	})
	palRGBA := image.NewPaletted(image.Rect(0, 0, 4, 1), color.Palette{
		color.RGBA{}, color.RGBA{66, 66, 66, 66}, color.RGBA{67, 67, 67, 67}, color.RGBA{255, 255, 255, 255},
	})
	copy(palNRGBA.Pix, []uint8{0, 1, 2, 3})
	copy(palRGBA.Pix, []uint8{0, 1, 2, 3})
	palShort := image.NewPaletted(image.Rect(0, 0, 2, 1), color.Palette{color.NRGBA{255, 255, 255, 255}})
	palShort.Pix[0], palShort.Pix[1] = 0, 200

	gray := image.NewGray(image.Rect(0, 0, 4, 1))
	gray16 := image.NewGray16(image.Rect(0, 0, 4, 1))
	rgba := image.NewRGBA(image.Rect(0, 0, 4, 1))
	rgba64 := image.NewRGBA64(image.Rect(0, 0, 4, 1))
	ycbcr := image.NewYCbCr(image.Rect(0, 0, 4, 1), image.YCbCrSubsampleRatio444)
	for i, v := range []uint8{0, 66, 67, 255} {
		gray.SetGray(i, 0, color.Gray{v})
		gray16.SetGray16(i, 0, color.Gray16{uint16(v) << 8})
		rgba.SetRGBA(i, 0, color.RGBA{v, v, v, v})
		rgba64.SetRGBA64(i, 0, color.RGBA64{uint16(v) << 8, uint16(v) << 8, uint16(v) << 8, uint16(v) << 8})
		ycbcr.Y[i], ycbcr.Cb[i], ycbcr.Cr[i] = v, 128, 128
	}

	tests := []struct {
		name string
		img  image.Image
		want []byte
	}{
		{"NRGBA", nrgba, []byte{0, 0, 1, 1, 0, 1}},
		{"NRGBA64", nrgba64, []byte{0, 0, 1, 1}},
		{"Paletted with NRGBA entries", palNRGBA, []byte{0, 0, 1, 1}},
		{"Paletted with RGBA entries", palRGBA, []byte{0, 0, 1, 1}},
		{"Paletted index past the palette", palShort, []byte{1, 0}},
		{"Gray", gray, []byte{0, 0, 1, 1}},
		{"Gray16", gray16, []byte{0, 0, 1, 1}},
		{"RGBA", rgba, []byte{0, 0, 1, 1}},
		{"RGBA64", rgba64, []byte{0, 0, 1, 1}},
		{"YCbCr", ycbcr, []byte{0, 0, 1, 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := newInkBitmap(tt.img, testThreshold)
			assert.Equal(t, len(tt.want), got.width)
			assert.Equal(t, 1, got.height)
			assert.Equal(t, tt.want, got.pixels)
		})
	}
}

func TestNewInkBitmapBounds(t *testing.T) {
	t.Parallel()
	whole := image.NewNRGBA(image.Rect(0, 0, 10, 12))
	whole.SetNRGBA(6, 8, color.NRGBA{255, 255, 255, 255})
	whole.SetNRGBA(7, 9, color.NRGBA{255, 255, 255, 255})
	sub := whole.SubImage(image.Rect(5, 7, 9, 11))

	moved := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	moved.SetNRGBA(1, 1, color.NRGBA{255, 255, 255, 255})
	moved.SetNRGBA(2, 2, color.NRGBA{255, 255, 255, 255})

	assert.Equal(t, newInkBitmap(moved, testThreshold), newInkBitmap(sub, testThreshold))
	assert.Equal(t, blank(0, 0), newInkBitmap(image.NewNRGBA(image.Rect(3, 3, 3, 8)), testThreshold))
}

func pngChunk(kind string, data []byte) []byte {
	out := binary.BigEndian.AppendUint32(nil, uint32(len(data)))
	out = append(out, kind...)
	out = append(out, data...)
	return binary.BigEndian.AppendUint32(out, crc32.ChecksumIEEE(append([]byte(kind), data...)))
}

type extraChunk struct {
	kind string
	data []byte
}

// buildPNG returns a PNG of one row. extra holds the PLTE or tRNS chunks.
func buildPNG(t *testing.T, width, depth, colourType int, extra []extraChunk, row []byte) []byte {
	t.Helper()
	ihdr := binary.BigEndian.AppendUint32(nil, uint32(width))
	ihdr = binary.BigEndian.AppendUint32(ihdr, 1)
	ihdr = append(ihdr, byte(depth), byte(colourType), 0, 0, 0)
	var idat bytes.Buffer
	zw := zlib.NewWriter(&idat)
	_, err := zw.Write(append([]byte{0}, row...))
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	out := []byte("\x89PNG\r\n\x1a\n")
	out = append(out, pngChunk("IHDR", ihdr)...)
	for _, e := range extra {
		out = append(out, pngChunk(e.kind, e.data)...)
	}
	out = append(out, pngChunk("IDAT", idat.Bytes())...)
	return append(out, pngChunk("IEND", nil)...)
}

func TestNewInkBitmapPNG(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		width int
		depth int
		ct    int
		extra []extraChunk
		row   []byte
		typ   image.Image
		want  []byte
	}{
		{"grey 1-bit", 4, 1, 0, nil, []byte{0b10100000}, &image.Gray{}, []byte{1, 0, 1, 0}},
		{
			"grey 8 with tRNS", 4, 8, 0,
			[]extraChunk{{"tRNS", []byte{0, 100}}},
			[]byte{255, 100, 70, 60},
			&image.NRGBA{},
			[]byte{1, 0, 1, 0},
		},
		{
			"grey 16", 4, 16, 0, nil,
			[]byte{0xFF, 0xFF, 0x20, 0, 0x55, 0, 0, 0},
			&image.Gray16{},
			[]byte{1, 0, 1, 0},
		},
		{
			"RGB 8 with tRNS", 4, 8, 2,
			[]extraChunk{{"tRNS", []byte{0, 10, 0, 10, 0, 10}}},
			[]byte{255, 255, 0, 10, 10, 10, 100, 100, 0, 60, 60, 60},
			&image.NRGBA{},
			[]byte{1, 0, 1, 0},
		},
		{
			"RGB 16", 4, 16, 2, nil,
			[]byte{0xFF, 0, 0xFF, 0, 0, 0, 0x10, 0, 0x10, 0, 0x10, 0, 0x64, 0, 0x64, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			&image.RGBA64{},
			[]byte{1, 0, 1, 0},
		},
		{
			"palette 2-bit with tRNS", 4, 2, 3,
			[]extraChunk{
				{"PLTE", []byte{255, 255, 255, 0, 0, 0, 255, 255, 255, 255, 255, 255}},
				{"tRNS", []byte{255, 255, 66, 67}},
			},
			[]byte{0b00011011},
			&image.Paletted{},
			[]byte{1, 0, 0, 1},
		},
		{
			"grey with alpha 16", 5, 16, 4, nil,
			[]byte{0xFF, 0xFF, 0x43, 0, 0xFF, 0xFF, 0x42, 0, 0xFF, 0xFF, 0, 0, 0, 0, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			&image.NRGBA64{},
			[]byte{1, 0, 0, 0, 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			img, err := png.Decode(bytes.NewReader(buildPNG(t, tt.width, tt.depth, tt.ct, tt.extra, tt.row)))
			require.NoError(t, err)
			assert.IsType(t, tt.typ, img)
			assert.Equal(t, tt.want, newInkBitmap(img, testThreshold).pixels)
		})
	}
}

func TestNewInkBitmapThreshold(t *testing.T) {
	t.Parallel()
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{255, 255, 255, 0x43})
	img.SetNRGBA(1, 0, color.NRGBA{0, 0, 0, 255})

	assert.Equal(t, []byte{1, 0}, newInkBitmap(img, 201).pixels)
	assert.Equal(t, []byte{0, 0}, newInkBitmap(img, 202).pixels)
	assert.Equal(t, []byte{1, 1}, newInkBitmap(img, 0).pixels)
}
