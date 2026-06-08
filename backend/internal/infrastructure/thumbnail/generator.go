// Package thumbnail implements thumbapp.Generator: it decodes, resizes
// (preserving aspect ratio, never upscaling) and re-encodes images under a
// bounded-concurrency semaphore to avoid memory spikes on small servers.
package thumbnail

import (
	"bytes"
	"context"
	"fmt"

	"github.com/disintegration/imaging"

	"scout/internal/application/thumbapp"
)

// Generator bounds concurrent image processing with a buffered-channel semaphore.
type Generator struct {
	sem chan struct{}
}

// NewGenerator builds a Generator allowing at most maxConcurrency simultaneous
// decode/resize/encode operations.
func NewGenerator(maxConcurrency int) *Generator {
	if maxConcurrency < 1 {
		maxConcurrency = 1
	}
	return &Generator{sem: make(chan struct{}, maxConcurrency)}
}

// Generate produces a thumbnail for the given original bytes and params.
func (g *Generator) Generate(ctx context.Context, original []byte, p thumbapp.Params) (thumbapp.Thumbnail, error) {
	// Bound concurrency; respect cancellation while waiting for a slot.
	select {
	case g.sem <- struct{}{}:
		defer func() { <-g.sem }()
	case <-ctx.Done():
		return thumbapp.Thumbnail{}, ctx.Err()
	}

	img, err := imaging.Decode(bytes.NewReader(original), imaging.AutoOrientation(true))
	if err != nil {
		return thumbapp.Thumbnail{}, fmt.Errorf("decode image: %w", err)
	}

	target := p.EffectiveWidth(img.Bounds().Dx())
	// imaging.Resize with height=0 preserves the aspect ratio (no cropping).
	dst := imaging.Resize(img, target, 0, imaging.Lanczos)

	var buf bytes.Buffer
	if err := imaging.Encode(&buf, dst, imaging.JPEG, imaging.JPEGQuality(p.Quality)); err != nil {
		return thumbapp.Thumbnail{}, fmt.Errorf("encode image: %w", err)
	}

	return thumbapp.Thumbnail{Data: buf.Bytes(), ContentType: p.ContentType()}, nil
}
