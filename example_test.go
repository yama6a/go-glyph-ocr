package glyphocr_test

import (
	"fmt"
	"image"
	"image/png"
	"os"

	glyphocr "github.com/yama6a/go-glyph-ocr"
)

func loadPGS04() image.Image {
	f, err := os.Open("testdata/pgs/04.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		panic(err)
	}

	return img
}

func ExampleRecognizer_Recognize() {
	r, err := glyphocr.NewRecognizer(glyphocr.SubtitleFonts())
	if err != nil {
		panic(err)
	}
	fmt.Println(r.Recognize(loadPGS04()).Text())
	// Output:
	// Hello, my friend.
	// How are you?
}

func ExampleTrainer_Train() {
	img := loadPGS04()
	empty, err := glyphocr.NewDatabase()
	if err != nil {
		panic(err)
	}
	r, err := glyphocr.NewRecognizer(empty)
	if err != nil {
		panic(err)
	}
	t, err := glyphocr.NewTrainer(glyphocr.DefaultLineCount, 0)
	if err != nil {
		panic(err)
	}

	letters := []rune("Hello,myfriend.Howareyou?")
	next := 0
	for _, c := range r.Recognize(img).UnknownChars() {
		g, err := t.Train(c.Sample, string(letters[next]), false)
		if err != nil {
			panic(err)
		}
		next++
		if err := empty.Add(g); err != nil {
			panic(err)
		}
	}

	r.Reset()
	fmt.Println(r.Recognize(img).Text())
	// Output:
	// Hello, my friend.
	// How are you?
}
