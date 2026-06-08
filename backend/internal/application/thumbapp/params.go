// Package thumbapp contains the thumbnail engine use case: parameter parsing
// and validation, deterministic cache keys, two-level caching, request
// deduplication (singleflight) and bounded image generation.
//
// The interfaces this package depends on (Cache, OriginalStore, Generator)
// are declared here in the consumer package.
package thumbapp

import (
	"context"
	"strconv"
)

// ParamError is a typed validation error for thumbnail parameters.
type ParamError struct {
	Field string
	Issue string
}

func (e ParamError) Error() string { return e.Field + ": " + e.Issue }

// Allowed parameter values. Declared as functions returning local slices to
// avoid package-level (global) variables.
func allowedWidths() []int    { return []int{320, 640, 960, 1280} }
func allowedDPRs() []int      { return []int{1, 2, 3} }
func allowedQualities() []int { return []int{70, 80, 90} }
func allowedFormats() []string {
	// webp is intentionally not implemented (pure-Go webp encoding is unreliable),
	// so only jpeg is accepted. Unsupported formats are rejected with 400.
	return []string{"jpeg"}
}

// Params are the validated thumbnail request parameters.
type Params struct {
	Width   int
	DPR     int
	Quality int
	Format  string
}

func containsInt(xs []int, v int) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func containsStr(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

// ParseParams validates raw query values and returns typed Params.
// Empty values fall back to sane defaults (width=320, dpr=1, quality=80, jpeg).
func ParseParams(width, dpr, quality, format string) (Params, error) {
	p := Params{Width: 320, DPR: 1, Quality: 80, Format: "jpeg"}

	if width != "" {
		w, err := strconv.Atoi(width)
		if err != nil || !containsInt(allowedWidths(), w) {
			return Params{}, ParamError{Field: "width", Issue: "must be one of 320, 640, 960, 1280"}
		}
		p.Width = w
	}
	if dpr != "" {
		d, err := strconv.Atoi(dpr)
		if err != nil || !containsInt(allowedDPRs(), d) {
			return Params{}, ParamError{Field: "dpr", Issue: "must be one of 1, 2, 3"}
		}
		p.DPR = d
	}
	if quality != "" {
		q, err := strconv.Atoi(quality)
		if err != nil || !containsInt(allowedQualities(), q) {
			return Params{}, ParamError{Field: "quality", Issue: "must be one of 70, 80, 90"}
		}
		p.Quality = q
	}
	if format != "" {
		if !containsStr(allowedFormats(), format) {
			return Params{}, ParamError{Field: "format", Issue: "must be jpeg"}
		}
		p.Format = format
	}
	return p, nil
}

// EffectiveWidth returns the target render width (width*dpr) capped by the
// original image width to avoid upscaling.
func (p Params) EffectiveWidth(originalWidth int) int {
	w := p.Width * p.DPR
	if w > originalWidth {
		return originalWidth
	}
	return w
}

// CacheKey builds the deterministic two-level cache key for these params.
func (p Params) CacheKey(photoID string) string {
	return "thumbnail:" + photoID +
		":w" + strconv.Itoa(p.Width) +
		":dpr" + strconv.Itoa(p.DPR) +
		":q" + strconv.Itoa(p.Quality) +
		":fmt" + p.Format
}

// ContentType returns the MIME type implied by the format.
func (p Params) ContentType() string {
	if p.Format == "webp" {
		return "image/webp"
	}
	return "image/jpeg"
}

// Thumbnail is a generated image with its content type.
type Thumbnail struct {
	Data        []byte
	ContentType string
}

// Cache is the two-level (LRU + Redis) thumbnail byte cache.
// Get reports which level served the value ("lru" or "redis").
type Cache interface {
	Get(ctx context.Context, key string) (Thumbnail, string, bool)
	Set(ctx context.Context, key string, t Thumbnail)
}

// OriginalStore fetches original image bytes from object storage.
// found is false when the object has not been uploaded yet.
type OriginalStore interface {
	GetOriginal(ctx context.Context, key string) (data []byte, found bool, err error)
}

// Generator resizes/encodes a thumbnail under bounded concurrency.
type Generator interface {
	Generate(ctx context.Context, original []byte, p Params) (Thumbnail, error)
}

// Metrics observes thumbnail engine activity. Implemented in infrastructure.
type Metrics interface {
	ThumbnailRequested()
	ThumbnailCacheHit(level string)
	ThumbnailCacheMiss()
	ThumbnailGenerated(seconds float64)
	ThumbnailGenerationError()
	GenerationStarted()
	GenerationFinished()
}
