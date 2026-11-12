// Package glyphocr reads the text of clean rendered text bitmaps, such as the images of
// Blu-ray and DVD subtitles. It is a port of php-glyph-ocr, which ports the nOCR engine
// of Subtitle Edit.
//
// The engine cuts an image into lines and glyphs and compares each glyph with a Database of
// known glyphs. It reads text that a computer drew, in a font that the database knows.
// SubtitleFonts and Latin return the embedded databases.
//
// Decode the image with image/png, build a Recognizer and call Recognize:
//
//	r, err := glyphocr.NewRecognizer(glyphocr.SubtitleFonts())
//	if err != nil {
//		return err
//	}
//	fmt.Println(r.Recognize(img).Text())
//
// A Recognizer learns the glyph heights from the images it reads. Use one Recognizer per
// subtitle stream, or call Reset. A Recognizer is not safe for concurrent use.
//
// A Trainer makes new glyphs from the samples of unknown characters, so a Database can learn
// a new font. The README has the training program, the accuracy and speed numbers, the
// differences from php-glyph-ocr and the attribution of the code and the font data:
// https://github.com/yama6a/go-glyph-ocr#readme
package glyphocr
