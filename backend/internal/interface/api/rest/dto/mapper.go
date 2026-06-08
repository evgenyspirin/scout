package dto

import (
	"time"

	"scout/internal/domain/photo"
)

// MapPhoto converts a domain Photo into its API DTO.
func MapPhoto(p photo.Photo, originalURL string) PhotoDTO {
	preds := make([]PredictionDTO, len(p.Predictions))
	for i, pr := range p.Predictions {
		preds[i] = PredictionDTO{
			ClassID:    pr.ClassID,
			Confidence: pr.Confidence,
			BBox: BBoxDTO{
				XMin: pr.BBox.XMin,
				YMin: pr.BBox.YMin,
				XMax: pr.BBox.XMax,
				YMax: pr.BBox.YMax,
			},
		}
	}
	return PhotoDTO{
		ID:          p.ID,
		X:           p.X,
		Y:           p.Y,
		H:           p.H,
		Width:       p.Width,
		Height:      p.Height,
		CapturedAt:  p.CapturedAt.UTC().Format(time.RFC3339),
		OriginalURL: originalURL,
		Predictions: preds,
	}
}
