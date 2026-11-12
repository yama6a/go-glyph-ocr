<?php

// Writes the PHP reference files for go-glyph-ocr. Run it with the PHP of php-glyph-ocr 0.3.0
// (commit 10e3eb3) after `composer dump-autoload --dev`, from the root of that checkout:
//   php export.php <php-glyph-ocr root> <go-glyph-ocr testdata dir>

use GlyphOcr\GlyphDatabase;
use GlyphOcr\Image;
use GlyphOcr\Recognizer;
use GlyphOcr\Tests\Fixtures\Generator\Fixtures;
use GlyphOcr\Tests\Fixtures\Generator\SubtitleFonts;

[, $root, $out] = $argv;
require "$root/vendor/autoload.php";

foreach (['latin' => 'latin', 'subtitle-fonts' => 'subtitleFonts'] as $file => $factory) {
    $database = GlyphDatabase::$factory();
    $recognizers = [];
    $lines = '';
    foreach (glob("$root/tests/fixtures/*/*.png") as $path) {
        $set = basename(dirname($path));
        $recognizers[$set] ??= new Recognizer($database);
        $text = $recognizers[$set]->recognize(Image::fromPng((string)file_get_contents($path)))->text();
        $lines .= "$set/" . basename($path) . ' | ' . str_replace("\n", ' / ', $text) . "\n";
    }
    file_put_contents("$out/php/$file.txt", $lines);
}

$dir = "$out/training/LiberationSans-Regular-28";
mkdir($dir, 0777, true);
file_put_contents("$dir/chars.txt", SubtitleFonts::CHARS . "\n");
foreach (mb_str_split(SubtitleFonts::CHARS) as $i => $char) {
    $spec = Fixtures::styled('bluray', ["Hd    $char"], 'LiberationSans-Regular.ttf', SubtitleFonts::SIZES[0]);
    file_put_contents(sprintf('%s/%03d.png', $dir, $i + 1), Fixtures::render($spec));
}
