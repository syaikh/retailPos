package escpos

// Builder accumulates ESC/POS bytes for a receipt.
type Builder struct {
	buf []byte
}

// NewBuilder returns an empty receipt builder.
func NewBuilder() *Builder { return &Builder{} }

// Write appends raw bytes to the builder and returns it for chaining.
func (b *Builder) Write(p []byte) *Builder {
	b.buf = append(b.buf, p...)
	return b
}

// Bytes returns the accumulated byte stream.
func (b *Builder) Bytes() []byte { return b.buf }
