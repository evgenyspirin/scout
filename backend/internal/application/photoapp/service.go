// Package photoapp contains photo-related use cases.
//
// Following Go convention, the interfaces this package depends on
// (PhotoRepository, ObjectStorage) are declared here, in the consumer
// package, not in the packages that implement them.
package photoapp

import (
	"context"
	"errors"
	"net/http"
	"time"

	"scout/internal/domain/photo"
)

// notFoundError signals a missing photo. A typed error is used instead of a
// package-level sentinel variable to comply with the no-global-variables rule.
type notFoundError struct{ id string }

func (e notFoundError) Error() string { return "photo not found: " + e.id }

// NotFound builds a not-found error for the given photo id.
func NotFound(id string) error { return notFoundError{id: id} }

// IsNotFound reports whether err signals a missing photo.
func IsNotFound(err error) bool {
	var nf notFoundError
	return errors.As(err, &nf)
}

// Filter selects which photos are returned by List.
type Filter struct {
	ClassID       string
	MinConfidence *float64
}

// Page is a cursor-paginated slice of photos.
type Page struct {
	Items     []photo.Photo
	NextToken string
}

// PhotoRepository reads photos and predictions from the dataset.
type PhotoRepository interface {
	List(ctx context.Context, f Filter, cursor string, limit int) (Page, error)
	GetByID(ctx context.Context, id string) (photo.Photo, error)
	Exists(ctx context.Context, id string) (bool, error)
}

// UploadLink describes a presigned PUT for an original image.
type UploadLink struct {
	URL       string
	Method    string
	Headers   map[string]string
	ExpiresAt time.Time
	Exists    bool
}

// ObjectStorage abstracts the object store used for originals.
type ObjectStorage interface {
	PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (string, map[string]string, time.Time, error)
	ObjectExists(ctx context.Context, key string) (bool, error)
}

// Service implements photo browsing and upload-link use cases.
type Service struct {
	repo          PhotoRepository
	storage       ObjectStorage
	presignTTL    time.Duration
	publicBaseURL string
}

// NewService builds a photo Service.
func NewService(repo PhotoRepository, storage ObjectStorage, presignTTL time.Duration, publicBaseURL string) *Service {
	return &Service{repo: repo, storage: storage, presignTTL: presignTTL, publicBaseURL: publicBaseURL}
}

// List returns a page of photos with all of their predictions.
func (s *Service) List(ctx context.Context, f Filter, cursor string, limit int) (Page, error) {
	return s.repo.List(ctx, f, cursor, limit)
}

// Get returns a single photo or a not-found error.
func (s *Service) Get(ctx context.Context, id string) (photo.Photo, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return photo.Photo{}, err
	}
	return p, nil
}

// OriginalURL builds the absolute URL the frontend uses to load the original.
// Originals are served through the backend (which streams them from object
// storage); this keeps the object store private and works behind any ingress.
func (s *Service) OriginalURL(id string) string {
	return s.publicBaseURL + "/api/v1/photos/" + id + "/original"
}

// CreateUploadLink returns a presigned PUT URL for a photo's original bytes.
// The response includes Exists so the seeder can stay idempotent.
func (s *Service) CreateUploadLink(ctx context.Context, id, contentType string) (UploadLink, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return UploadLink{}, err
	}
	key := p.ObjectKey()
	exists, err := s.storage.ObjectExists(ctx, key)
	if err != nil {
		return UploadLink{}, err
	}
	url, headers, expires, err := s.storage.PresignPut(ctx, key, contentType, s.presignTTL)
	if err != nil {
		return UploadLink{}, err
	}
	return UploadLink{
		URL:       url,
		Method:    http.MethodPut,
		Headers:   headers,
		ExpiresAt: expires,
		Exists:    exists,
	}, nil
}

// ObjectExists reports whether a photo's original is present in storage.
func (s *Service) ObjectExists(ctx context.Context, id string) (bool, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return false, err
	}
	return s.storage.ObjectExists(ctx, p.ObjectKey())
}
