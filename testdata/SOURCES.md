# Sources

Every image here was drawn for php-glyph-ocr by its fixture generator. No image holds text from a film.
The golden images and their texts are copied unchanged from [php-glyph-ocr](https://github.com/yama6a/php-glyph-ocr) at commit `10e3eb3`, under the MIT licence of that repository.
No font file is copied. The font licences govern only the font files.

## Golden images

Each `<set>/<nn>.png` has a `<set>/<nn>.txt` with the exact text, one line per text line.

| Set | Shape | Fonts and sizes |
|:--- |:--- |:--- |
| `bluray` | 8-bit RGBA, smooth edges, white or yellow fill, black outline of size / 14 pixels | DejaVu Sans, Liberation Sans, 36 to 60 px |
| `pgs` | Cropped object bitmap as a Blu-ray PGS stream holds it: a palette of up to 256 RGBA entries, white fill, black outline | DejaVu Sans, Liberation Sans, 40 to 60 px |
| `dvd` | 720 pixels wide, 4 palette colours as a DVD VobSub subpicture: transparent, white fill, black outline, grey edge | DejaVu Sans, Liberation Sans, 24 to 30 px |
| `italic` | As `bluray`, with the italic font files | DejaVu Sans Oblique, Liberation Sans Italic, 40 to 52 px |
| `plain` | White text without outline, 2 with smooth edges and 2 without anti-aliasing | DejaVu Sans, Liberation Sans, 30 to 40 px |
| `small` | As `bluray`, with a 1.5 pixel outline | DejaVu Sans, Liberation Sans, 20 to 24 px |
| `capital-i` | Lines with capital I and lower case l, in the shapes of the sets above | DejaVu Sans, Liberation Sans, Noto Sans and their italics, Open Sans, 22 to 52 px |
| `unseen` | General lines in the shapes of the sets above, in a font that no bundled database has glyphs of | Open Sans, Open Sans Italic, 24 to 48 px |

## Fonts

| Font | Source | Licence |
|:--- |:--- |:--- |
| DejaVu Sans, DejaVu Sans Oblique | [DejaVu Fonts 2.37](https://github.com/dejavu-fonts/dejavu-fonts/releases/tag/version_2_37) | Bitstream Vera licence with public domain changes |
| Liberation Sans Regular, Italic | [Liberation Fonts 2.1.5](https://github.com/liberationfonts/liberation-fonts/releases/tag/2.1.5) | SIL Open Font Licence 1.1 |
| Noto Sans Regular, Italic | [Noto Sans 2.015](https://github.com/notofonts/latin-greek-cyrillic/releases/tag/NotoSans-v2.015), unhinted files | SIL Open Font Licence 1.1 |
| Open Sans Regular, Italic | [Open Sans](https://github.com/googlefonts/opensans/tree/bd7e37632246368c60fdcbd374dbf9bad11969b6/fonts/ttf) at commit `bd7e376` | SIL Open Font Licence 1.1 |

## Subtitle Edit reads

`subtitle-edit/expected.txt` holds what the nOCR engine of Subtitle Edit at commit `b8da12a` reads from the 51 images with `Latin.nocr`.
It is copied unchanged from php-glyph-ocr at `10e3eb3`.
The .NET test program that wrote it is [tests/fixtures/subtitle-edit](https://github.com/yama6a/php-glyph-ocr/tree/10e3eb3832d2b9fb70ddc6591cfe4d6d1da9d5ed/tests/fixtures/subtitle-edit). It is not copied, because the images are the same.

## PHP export

`php/export.php` ran once on php-glyph-ocr 0.3.0 at `10e3eb3`, with PHP 8.5.12 and `composer dump-autoload --dev`.
It wrote these files, and nobody edits them:

- `php/latin.txt` and `php/subtitle-fonts.txt`: the text of every image read by php-glyph-ocr with default options, one recognizer per set, images in path order. A line break shows as ` / `.
- `training/LiberationSans-Regular-28/001.png` to `109.png`: the 109 renderings of `Hd    <char>` in Liberation Sans Regular, bluray style, 28 px, in the order of `chars.txt`. The trainer parity test rebuilds one block of `resources/SubtitleFonts.nocr` from them.

## Go goldens

`golden/latin.txt` and `golden/subtitle-fonts.txt` hold the reads of this module in the format of the PHP export. `make golden` rewrites them.
`TestPHPReads` compares them with the PHP export. A line may differ only when `phpDifferences` in `parity_test.go` names it with a reason.
