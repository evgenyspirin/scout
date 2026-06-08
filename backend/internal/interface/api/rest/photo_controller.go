package rest

import (
	"context"
	"io"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"scout/internal/application/photoapp"
	"scout/internal/interface/api/rest/dto"
	"scout/internal/interface/api/rest/validator"
)

// OriginalStreamer streams original image bytes from object storage.
// Declared here in the consumer package per Go convention.
type OriginalStreamer interface {
	StreamOriginal(ctx context.Context, key string) (io.ReadCloser, string, int64, bool, error)
}

// PhotoController handles photo browsing and upload-link endpoints.
type PhotoController struct {
	logger   *zap.Logger
	photos   *photoapp.Service
	streamer OriginalStreamer
}

// NewPhotoController builds a PhotoController.
func NewPhotoController(logger *zap.Logger, photos *photoapp.Service, streamer OriginalStreamer) *PhotoController {
	return &PhotoController{logger: logger, photos: photos, streamer: streamer}
}

const defaultListLimit = 50

// List returns a cursor-paginated page of photos with their predictions.
func (p *PhotoController) List(c *fiber.Ctx) error {
	limit := defaultListLimit
	if raw := c.Query("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return NewValidation("invalid query", []FieldError{{Field: "limit", Issue: "must be an integer"}})
		}
		limit = v
	}

	var minConf *float64
	if raw := c.Query("minConfidence"); raw != "" {
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return NewValidation("invalid query", []FieldError{{Field: "minConfidence", Issue: "must be a number"}})
		}
		minConf = &v
	}

	if errs := validator.ValidateListQuery(limit, minConf); len(errs) > 0 {
		return NewValidation("invalid query", toFieldErrors(errs))
	}

	filter := photoapp.Filter{ClassID: c.Query("classId"), MinConfidence: minConf}
	page, err := p.photos.List(c.Context(), filter, c.Query("cursor"), limit)
	if err != nil {
		return err
	}

	items := make([]dto.PhotoDTO, len(page.Items))
	for i, ph := range page.Items {
		items[i] = dto.MapPhoto(ph, p.photos.OriginalURL(ph.ID))
	}
	return writeJSON(c, fiber.StatusOK, &dto.PhotoPageDTO{Items: items, NextToken: page.NextToken})
}

// Get returns a single photo with all its predictions.
func (p *PhotoController) Get(c *fiber.Ctx) error {
	id := c.Params("photoId")
	ph, err := p.photos.Get(c.Context(), id)
	if err != nil {
		if photoapp.IsNotFound(err) {
			return NewNotFound("photo not found", id)
		}
		return err
	}
	dtoPhoto := dto.MapPhoto(ph, p.photos.OriginalURL(ph.ID))
	return writeJSON(c, fiber.StatusOK, &dtoPhoto)
}

// CreateUploadLink returns a presigned PUT URL for the original (admin only).
func (p *PhotoController) CreateUploadLink(c *fiber.Ctx) error {
	id := c.Params("photoId")
	var req dto.UploadLinkRequest
	if err := c.BodyParser(&req); err != nil {
		return NewBadRequest("invalid JSON body")
	}
	if errs := validator.ValidateUploadLink(req); len(errs) > 0 {
		return NewValidation("invalid request body", toFieldErrors(errs))
	}
	link, err := p.photos.CreateUploadLink(c.Context(), id, req.ContentType)
	if err != nil {
		if photoapp.IsNotFound(err) {
			return NewNotFound("photo not found", id)
		}
		return err
	}
	return writeJSON(c, fiber.StatusOK, &dto.UploadLinkDTO{
		URL:       link.URL,
		Method:    link.Method,
		Headers:   link.Headers,
		ExpiresAt: link.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		Exists:    link.Exists,
	})
}

// HeadObject reports object existence for seeder idempotency (admin only).
func (p *PhotoController) HeadObject(c *fiber.Ctx) error {
	id := c.Params("photoId")
	exists, err := p.photos.ObjectExists(c.Context(), id)
	if err != nil {
		if photoapp.IsNotFound(err) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return err
	}
	c.Set("X-Object-Exists", strconv.FormatBool(exists))
	if !exists {
		return c.SendStatus(fiber.StatusNotFound)
	}
	return c.SendStatus(fiber.StatusOK)
}

// Original streams the original image from object storage.
func (p *PhotoController) Original(c *fiber.Ctx) error {
	id := c.Params("photoId")
	reader, contentType, size, found, err := p.streamer.StreamOriginal(c.Context(), id+".jpg")
	if err != nil {
		return err
	}
	if !found {
		return NewNotFound("original image not found", id)
	}
	c.Set(fiber.HeaderContentType, contentType)
	c.Set(fiber.HeaderCacheControl, "public, max-age=86400")
	return c.SendStream(reader, int(size))
}
