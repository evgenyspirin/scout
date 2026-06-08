package thumbapp

import (
	"context"
	"errors"
	"time"

	"golang.org/x/sync/singleflight"
)

// originalNotFoundError signals the original image has not been uploaded.
type originalNotFoundError struct{ id string }

func (e originalNotFoundError) Error() string { return "original not found for photo: " + e.id }

// IsOriginalNotFound reports whether err means the original is missing.
func IsOriginalNotFound(err error) bool {
	var e originalNotFoundError
	return errors.As(err, &e)
}

// Service orchestrates cache lookup, deduplicated generation and metrics.
type Service struct {
	cache   Cache
	store   OriginalStore
	gen     Generator
	metrics Metrics
	sf      singleflight.Group
}

// NewService builds a thumbnail Service.
func NewService(cache Cache, store OriginalStore, gen Generator, metrics Metrics) *Service {
	return &Service{cache: cache, store: store, gen: gen, metrics: metrics}
}

// Get returns a cached or freshly generated thumbnail for the photo.
// Identical concurrent requests share a single generation via singleflight.
func (s *Service) Get(ctx context.Context, photoID string, p Params) (Thumbnail, error) {
	s.metrics.ThumbnailRequested()
	key := p.CacheKey(photoID)

	if t, level, ok := s.cache.Get(ctx, key); ok {
		s.metrics.ThumbnailCacheHit(level)
		return t, nil
	}
	s.metrics.ThumbnailCacheMiss()

	v, err, _ := s.sf.Do(key, func() (interface{}, error) {
		// Re-check the cache: a concurrent leader may have just populated it.
		if t, level, ok := s.cache.Get(ctx, key); ok {
			s.metrics.ThumbnailCacheHit(level)
			return t, nil
		}

		original, found, err := s.store.GetOriginal(ctx, photoID+".jpg")
		if err != nil {
			return Thumbnail{}, err
		}
		if !found {
			return Thumbnail{}, originalNotFoundError{id: photoID}
		}

		s.metrics.GenerationStarted()
		defer s.metrics.GenerationFinished()

		start := time.Now()
		t, genErr := s.gen.Generate(ctx, original, p)
		if genErr != nil {
			s.metrics.ThumbnailGenerationError()
			return Thumbnail{}, genErr
		}
		s.metrics.ThumbnailGenerated(time.Since(start).Seconds())

		s.cache.Set(ctx, key, t)
		return t, nil
	})
	if err != nil {
		return Thumbnail{}, err
	}
	return v.(Thumbnail), nil
}
