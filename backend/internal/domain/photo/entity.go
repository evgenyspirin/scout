// Package photo contains the Photo aggregate and its business rules.
// Domain entities are storage- and transport-agnostic.
package photo

import "time"

// BoundingBox is a detection box normalized to [0,1] against the original image.
// (XMin,YMin) is the top-left corner, (XMax,YMax) the bottom-right corner.
type BoundingBox struct {
	XMin float64
	YMin float64
	XMax float64
	YMax float64
}

// Width returns the normalized box width in [0,1].
func (b BoundingBox) Width() float64 { return b.XMax - b.XMin }

// Height returns the normalized box height in [0,1].
func (b BoundingBox) Height() float64 { return b.YMax - b.YMin }

// Prediction is a single model detection attached to a photo.
type Prediction struct {
	ID         string
	PhotoID    string
	ClassID    string
	Confidence float64
	BBox       BoundingBox
}

// Photo is a single greenhouse capture with its position and predictions.
type Photo struct {
	ID          string
	X           float64
	Y           float64
	H           float64
	Width       int
	Height      int
	CapturedAt  time.Time
	Predictions []Prediction
}

// ObjectKey returns the object-storage key for the photo's original bytes.
func (p Photo) ObjectKey() string { return p.ID + ".jpg" }
