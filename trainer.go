package glyphocr

// Trainer turns a confirmed sample into a database glyph.
type Trainer struct{}

// NewTrainer returns a trainer that makes lineCount line segments of each kind per glyph.
// The seed makes the choice of segments repeatable. DefaultLineCount is the value of Subtitle Edit.
func NewTrainer(_ int, _ uint32) (*Trainer, error) {
	panic("glyphocr: NewTrainer is not implemented")
}

// Train returns the glyph that describes s as the character text.
func (*Trainer) Train(_ *Sample, _ string, _ bool) (*Glyph, error) {
	panic("glyphocr: Trainer.Train is not implemented")
}
