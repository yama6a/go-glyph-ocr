# go-glyph-ocr

Pure Go OCR for clean rendered text bitmaps such as image subtitles. A port of php-glyph-ocr and of the nOCR engine of Subtitle Edit.

The engine cuts an image into lines and glyphs and compares each glyph with a database of known glyphs. It reads text that a computer drew, in a font that the database knows. It does not read photos, scans or handwriting.

## Install

```sh
go get github.com/yama6a/go-glyph-ocr
```

Needs Go 1.27 or later. No cgo and no dependency beyond the standard library.

## Usage

```go
f, err := os.Open("subtitle.png")
if err != nil {
	return err
}
defer f.Close()
img, err := png.Decode(f)
if err != nil {
	return err
}

r, err := glyphocr.NewRecognizer(glyphocr.SubtitleFonts())
if err != nil {
	return err
}
res := r.Recognize(img)
fmt.Println(res.Text())          // all lines, joined with "\n"
fmt.Println(res.Confidence())    // mean confidence of all glyphs, from 0 to 1
fmt.Println(res.Lines[0].Text()) // the first line
```

- **Images**: any `image.Image`. `image/png` decodes every PNG colour type. For untrusted files, check the size with `png.DecodeConfig` before `png.Decode`. The module sets no size limit.
- **Confidence**: the share of the database glyph's line points that agree with the image, from 0 to 1. An unknown glyph reads as `*` with confidence 0.
- **Ink**: a pixel is ink when the sum of its premultiplied red, green and blue is at least the ink threshold, 200 by default. So white or yellow text counts and a black outline does not.
- **State**: a recognizer learns the glyph heights from the images it reads. Use one recognizer per subtitle stream, or call `Reset`.
- **Concurrency**: a recognizer is not safe for concurrent use. A database is safe for concurrent reads while nobody calls `Add` or `Remove`.

Pass options to `NewRecognizer` after the database:

| Option | Default | Meaning |
|:--- |:--- |:--- |
| `InkThreshold(sum)` | `200` | Least premultiplied red + green + blue of an ink pixel, 1 to 765 |
| `SpaceWidth(columns)` | derived per line | Empty columns that make a space. Without it, a third of the typical glyph height of the line |
| `MaxWrongPixels(n)` | `25` | Error budget of the loose match passes |
| `KeepLatinCase()` | height fix on | Turns off the pick of upper or lower case for letters such as o and O from their height |
| `UnknownText(text)` | `"*"` | Text of a glyph that matches nothing. May be empty |
| `ItalicSlant(factor)` | `0` | Above 0, a glyph that matches nothing is slanted back by this factor and tried again. 0 to 1 |
| `RightToLeft()` | off | Puts the glyphs of each line in right to left order |
| `MinLineHeight(pixels)` | `12` | Minimum line height until the recognizer has learned the glyph heights |
| `NoLineContext()` | context on | Keeps the database text of each glyph, as Subtitle Edit does. By default the recognizer compares each glyph with the other glyphs of its line. It picks I or l, o, O or 0, and comma or apostrophe |

## Databases and training

`Database` reads and writes the `.nocr` files of Subtitle Edit, version 1 and 2. The module embeds two databases:

| Function | Glyphs | File size | Load time | Allocated per load |
|:--- | ---:| ---:| ---:| ---:|
| `SubtitleFonts()` | 2,007 | 254 KB + 477 KB | 19 ms | 12.1 MB |
| `Latin()` | 699 | 477 KB | 11 ms | 7.6 MB |

`BenchmarkSubtitleFonts` and `BenchmarkLatin` measure the load. Each parses the embedded files, and the `SubtitleFonts()` load parses both files. The first call of `SubtitleFonts()` or `Latin()` in a process does this work. Later calls copy the parsed database.

- **`SubtitleFonts()`**: glyphs trained on DejaVu Sans, Liberation Sans and Noto Sans, upright and italic, at 28 and 48 px, followed by the Latin database for other fonts. Liberation Sans has the metrics of Arial. Use this database for subtitles.
- **`Latin()`**: the Latin database of Subtitle Edit, unchanged. Use it to get the same result as Subtitle Edit.
- Other scripts and other fonts need their own database.

Train a glyph from a sample that a person confirmed. The new glyph goes first, so it wins over older glyphs that match equally well:

```go
db, err := glyphocr.LoadDatabase("my-font.nocr")
if err != nil {
	return err
}
r, err := glyphocr.NewRecognizer(db)
if err != nil {
	return err
}
t, err := glyphocr.NewTrainer(glyphocr.DefaultLineCount, 0)
if err != nil {
	return err
}
for _, c := range r.Recognize(img).UnknownChars() {
	g, err := t.Train(c.Sample, askAPerson(c.Sample.String()), false)
	if err != nil {
		return err
	}
	if err := db.Add(g); err != nil {
		return err
	}
}
return db.Save("my-font.nocr")
```

- `Recognizer.Split` returns the glyph samples of each line without matching them, with `nil` for a space.
- `MergeSamples` joins the parts of a character that the splitter cuts apart, such as `"` or `%`.
- The trainer draws random line segments with a seeded generator, so the same seed and the same samples in the same order give the same glyphs. With the same seed it gives the glyphs of php-glyph-ocr.

## Accuracy

The golden images in `testdata` are white or yellow subtitles, 20 to 60 pixels, 1 or 2 lines. `testdata/SOURCES.md` describes them. One recognizer reads each set in order.

Character accuracy is 1 minus the edit distance divided by the length of the expected text. The number of images read without error is in brackets. `GLYPHOCR_ACCURACY=1 go test -run TestAccuracyTable -v .` prints the numbers.

| Set | Fonts | `SubtitleFonts()` | `Latin()` |
|:--- |:--- | ---:| ---:|
| Blu-ray, smooth RGBA | DejaVu Sans, Liberation Sans | 98.8% (6 of 8) | 93.8% (1 of 8) |
| PGS palette | DejaVu Sans, Liberation Sans | 100% (6 of 6) | 93.0% (1 of 6) |
| DVD, 4 colours, 24 to 30 px | DejaVu Sans, Liberation Sans | 95.5% (3 of 6) | 63.2% (0 of 6) |
| Italic | DejaVu Sans, Liberation Sans | 98.0% (3 of 4) | 92.1% (1 of 4) |
| No outline | DejaVu Sans, Liberation Sans | 98.9% (3 of 4) | 79.8% (0 of 4) |
| Small, 20 to 24 px | DejaVu Sans, Liberation Sans | 98.1% (2 of 3) | 50.0% (0 of 3) |
| Capital I and lower case l | DejaVu Sans, Liberation Sans, Noto Sans, Open Sans | 99.4% (12 of 14) | 90.2% (2 of 14) |
| Unseen font | Open Sans | 96.7% (3 of 6) | 84.9% (0 of 6) |
| All | | 98.3% (38 of 51) | 85.0% (5 of 51) |

- The Latin database has no glyphs of these fonts. The numbers show how it does on fonts it has not seen. The subtitle fonts database has no glyphs of Open Sans.
- With `NoLineContext()`, the Latin database reads 80.3% of the characters (1 of 51 images) and the subtitle fonts database reads 91.2% (7 of 51).
- With `ItalicSlant(0.2)`, the Latin database reads 96.0% of the characters of the italic set (2 of 4 images).
- With the same database and `NoLineContext()`, the port reads every golden image exactly as Subtitle Edit does. `TestSubtitleEditParity` checks this.

## Speed

`BenchmarkRecognize` reads the 51 golden images one after another, with empty match caches. Every image is a cold read. The times are per image, after the database is loaded:

| Database | Time per image | Time per pass over 51 images |
|:--- | ---:| ---:|
| `SubtitleFonts()` | 7 ms | 0.36 s |
| `Latin()` | 18 ms | 0.94 s |
| Database that matches nothing | 37 ms | 1.90 s |

- A glyph that matches nothing is the slow case, because it runs through every match pass. So the subtitle fonts database is faster on the fonts it knows.
- The third row uses the Latin glyphs with the ink and background segments swapped, so almost no glyph matches.
- Crop the image to the text where you can. The splitter and the matcher work on every pixel of the image.
- Machine: Intel Core i5-8400T at 1.7 GHz, `GOMAXPROCS=2`, Go 1.27.4, linux/amd64.
- Command: `go test -run '^$' -bench . -benchmem -benchtime 5x .`. `make bench BENCHTIME=5x` runs the same command.

## Limits

- One text colour on a transparent or dark background. The ink threshold removes the outline, so text with a light outline or a light background does not work.
- The matcher compares shapes. Most sans-serif fonts draw I and l as the same bar, and l is about 5% taller. The recognizer picks the letter from the tops of the capitals, the ascenders and the other bars on the line. A line without them falls back to the heights of earlier lines, then to the neighbouring letters.
- No dictionary and no language model. Subtitle Edit fixes common OCR errors in a separate step, which this module does not port.
- Italic text needs italic glyphs in the database, or `ItalicSlant`.

## Errors

The functions wrap these sentinel errors. Test them with `errors.Is`.

| Error | Returned for |
|:--- |:--- |
| `ErrInvalidOption` | An option value outside its range, a missing database in `NewRecognizer`, a trainer line count outside 1 to 255 |
| `ErrInvalidGlyph` | A glyph that fails `Glyph.Validate`, in `NewDatabase`, `Database.Add` and `Trainer.Train` |
| `ErrInvalidSample` | A sample with a wrong pixel count or part count, in `Trainer.Train` and `MergeSamples` |
| `ErrInvalidDatabase` | Data that is not gzip-compressed, in `ReadDatabase` and `LoadDatabase` |

A damaged record in the gzip stream ends the read without an error, as in Subtitle Edit. The database then holds the glyphs before that record.

`LoadDatabase` of a path that does not exist returns the file error, so `errors.Is(err, fs.ErrNotExist)` is true.

## Differences from php-glyph-ocr

The port gives the same reads as php-glyph-ocr 0.3.0, except where the PHP code is wrong or where Go has other types or another API.

| Behaviour | PHP | Go | PHP issue |
|:--- |:--- |:--- |:--- |
| Trainer and a sample pixel of 2 | Counts it as neither ink nor background | Counts any non-zero pixel as ink | [php-glyph-ocr#10](https://github.com/yama6a/php-glyph-ocr/issues/10) |
| `ItalicSlant` of NaN | Accepted | Returns `ErrInvalidOption` | [php-glyph-ocr#11](https://github.com/yama6a/php-glyph-ocr/issues/11) |
| Save of a glyph wider than 1920 or higher than 1080 | Writes a file that the reader then cuts short | `Glyph.Validate` rejects the glyph, so every saved file reads back whole | [php-glyph-ocr#12](https://github.com/yama6a/php-glyph-ocr/issues/12) |
| Neighbour of a capital I, a letter plus a final newline | The pattern ends in `$`, which also matches before a final newline, so it counts the letter as a whole word | A word is exactly one code point | [php-glyph-ocr#13](https://github.com/yama6a/php-glyph-ocr/issues/13) |
| Right-to-left lines | No derived spaces | Derives spaces as for left-to-right lines | [php-glyph-ocr#14](https://github.com/yama6a/php-glyph-ocr/issues/14) |
| Right-to-left lines and multi-part glyphs such as `"` and `%` | Never matches them | Matches them | [php-glyph-ocr#15](https://github.com/yama6a/php-glyph-ocr/issues/15) |
| Images and PNG decoding | Own image classes and PNG decoder | `image.Image` and `image/png` | none, Go types |
| `Sample` X and Y | Count from the first ink row | Image coordinates. No read depends on them | none, Go API |
| `LoadDatabase` of a missing file | Throws a database exception | Returns the file error | none, Go types |

## Attribution

This module is a port of [php-glyph-ocr](https://github.com/yama6a/php-glyph-ocr) 0.3.0 (commit `10e3eb3`, MIT, yama6a). php-glyph-ocr is a port of the nOCR engine of [Subtitle Edit](https://github.com/SubtitleEdit/subtitleedit) by Nikolaj Olsson, MIT, at commit [`b8da12a`](https://github.com/SubtitleEdit/subtitleedit/tree/b8da12a4262e0294b1c8a67905b3cfe5066c1e00). `LICENSE` keeps both copyright notices.

| Go file | Subtitle Edit file in `src/libuilogic/Ocr` |
|:--- |:--- |
| `match.go`, `glyph.go` | `NOcrDb.cs`, `NOcrChar.cs` |
| `nocr.go`, `database.go` | `NOcrDb.cs`, `NOcrChar.cs` (file format) |
| `split.go` | `NikseBitmapImageSplitter2.cs`, `ImageSplitterItem2.cs` |
| `bitmap.go` | `NikseBitmap2.cs` |
| `linepoints.go` | `NOcrLine.cs` |
| `casefix.go` | `NOcrCaseFixer.cs` |
| `lineheight.go` | `OcrLineHeightTracker.cs` |
| `trainer.go` | `NOcrChar.cs`, `NOcrLineGenerator.cs` |
| `sample.go` (`MergeSamples`) | `ExpandedOcrGroup.cs` |
| `resources/Latin.nocr` | `Ocr/Latin.nocr` at the repository root, unchanged |

`spaces.go`, `linecontext.go`, `recognizer.go` and `image.go` are php-glyph-ocr code.

`resources/SubtitleFonts.nocr` is trained on renderings of these fonts. It holds line segments measured on rendered glyphs, and no font outlines. php-glyph-ocr builds it with `tests/fixtures/generator/train.php`. This module copies it unchanged. The test images in `testdata` use these fonts and Open Sans. `testdata/SOURCES.md` lists them.

| Font | Licence |
|:--- |:--- |
| [DejaVu Sans 2.37](https://github.com/dejavu-fonts/dejavu-fonts/releases/tag/version_2_37) | Bitstream Vera licence with public domain changes |
| [Liberation Sans 2.1.5](https://github.com/liberationfonts/liberation-fonts/releases/tag/2.1.5) | SIL Open Font Licence 1.1 |
| [Noto Sans 2.015](https://github.com/notofonts/latin-greek-cyrillic/releases/tag/NotoSans-v2.015) | SIL Open Font Licence 1.1 |
| [Open Sans](https://github.com/googlefonts/opensans/tree/bd7e37632246368c60fdcbd374dbf9bad11969b6/fonts/ttf), test images only | SIL Open Font Licence 1.1 |

The Mersenne Twister of `mt19937.go` follows the published algorithm of Makoto Matsumoto and Takuji Nishimura. The range rule follows the behaviour of PHP `Random\Randomizer::getInt`. The file copies no code of either.

## Development

```sh
make ci      # tidy, generate, format, lint, vet, test and vulnerability checks
make test    # go test with the race detector
make golden  # write the golden files again, then read the diff
make fuzz    # every fuzz target for FUZZTIME, 30 s by default
make bench   # every benchmark
```

Every merge to `main` with a `major`, `minor` or `patch` label is a release. The `skip-release` label releases nothing.
