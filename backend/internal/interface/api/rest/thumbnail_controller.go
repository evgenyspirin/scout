package rest

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"scout/internal/application/thumbapp"
)

// ThumbnailController serves on-demand thumbnails.
type ThumbnailController struct {
	logger      *zap.Logger
	thumbnails  *thumbapp.Service
	cacheMaxAge int
}

// NewThumbnailController builds a ThumbnailController.
func NewThumbnailController(logger *zap.Logger, thumbnails *thumbapp.Service, cacheMaxAge int) *ThumbnailController {
	return &ThumbnailController{logger: logger, thumbnails: thumbnails, cacheMaxAge: cacheMaxAge}
}

// Get returns a generated/cached thumbnail.
func (t *ThumbnailController) Get(c *fiber.Ctx) error {
	id := c.Params("photoId")
	params, err := thumbapp.ParseParams(
		c.Query("width"), c.Query("dpr"), c.Query("quality"), c.Query("format"),
	)
	if err != nil {
		if pe, ok := err.(thumbapp.ParamError); ok {
			return NewValidation("invalid thumbnail parameters", []FieldError{{Field: pe.Field, Issue: pe.Issue}})
		}
		return NewBadRequest(err.Error())
	}

	thumb, err := t.thumbnails.Get(c.Context(), id, params)
	if err != nil {
		if thumbapp.IsOriginalNotFound(err) {
			return NewNotFound("original image not found; upload it first", id)
		}
		return err
	}

	c.Set(fiber.HeaderContentType, thumb.ContentType)
	c.Set(fiber.HeaderCacheControl, "public, max-age="+strconv.Itoa(t.cacheMaxAge))
	c.Set(fiber.HeaderETag, params.CacheKey(id))
	return c.Send(thumb.Data)
}
