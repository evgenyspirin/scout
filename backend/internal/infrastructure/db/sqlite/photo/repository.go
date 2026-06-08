// Package sqlitephoto implements photoapp.PhotoRepository over SQLite.
package sqlitephoto

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"scout/internal/application/photoapp"
	"scout/internal/domain/photo"
)

// Repository reads photos and predictions from the dataset DB.
type Repository struct {
	db *sql.DB
}

// NewRepository builds a Repository.
func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

const (
	colsPhotos = "id, x, y, h, width, height, captured_at"
	colsPreds  = "id, photo_id, class_id, confidence, bbox_xmin, bbox_ymin, bbox_xmax, bbox_ymax"
)

func encodeCursor(id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(id))
}

func decodeCursor(token string) (string, error) {
	if token == "" {
		return "", nil
	}
	b, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", fmt.Errorf("invalid cursor")
	}
	return string(b), nil
}

// List returns a page of photos (each with ALL its predictions) matching the
// filter, using id-based cursor pagination.
func (r *Repository) List(ctx context.Context, f photoapp.Filter, cursor string, limit int) (photoapp.Page, error) {
	afterID, err := decodeCursor(cursor)
	if err != nil {
		return photoapp.Page{}, err
	}

	var conds []string
	var args []any
	if afterID != "" {
		conds = append(conds, "id > ?")
		args = append(args, afterID)
	}
	if f.ClassID != "" || f.MinConfidence != nil {
		sub := "EXISTS (SELECT 1 FROM predictions pr WHERE pr.photo_id = photos.id"
		if f.ClassID != "" {
			sub += " AND pr.class_id = ?"
			args = append(args, f.ClassID)
		}
		if f.MinConfidence != nil {
			sub += " AND pr.confidence >= ?"
			args = append(args, *f.MinConfidence)
		}
		sub += ")"
		conds = append(conds, sub)
	}

	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}
	query := "SELECT " + colsPhotos + " FROM photos" + where + " ORDER BY id ASC LIMIT ?"
	args = append(args, limit+1)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return photoapp.Page{}, fmt.Errorf("list photos: %w", err)
	}
	defer rows.Close()

	photos := make([]photo.Photo, 0, limit+1)
	for rows.Next() {
		p, scanErr := scanPhoto(rows)
		if scanErr != nil {
			return photoapp.Page{}, scanErr
		}
		photos = append(photos, p)
	}
	if err := rows.Err(); err != nil {
		return photoapp.Page{}, err
	}

	page := photoapp.Page{}
	if len(photos) > limit {
		page.NextToken = encodeCursor(photos[limit-1].ID)
		photos = photos[:limit]
	}

	if err := r.attachPredictions(ctx, photos); err != nil {
		return photoapp.Page{}, err
	}
	page.Items = photos
	return page, nil
}

// GetByID returns one photo with all its predictions.
func (r *Repository) GetByID(ctx context.Context, id string) (photo.Photo, error) {
	row := r.db.QueryRowContext(ctx, "SELECT "+colsPhotos+" FROM photos WHERE id = ?", id)
	p, err := scanPhoto(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return photo.Photo{}, photoapp.NotFound(id)
		}
		return photo.Photo{}, err
	}
	one := []photo.Photo{p}
	if err := r.attachPredictions(ctx, one); err != nil {
		return photo.Photo{}, err
	}
	return one[0], nil
}

// Exists reports whether a photo id is present in the dataset.
func (r *Repository) Exists(ctx context.Context, id string) (bool, error) {
	var found int
	err := r.db.QueryRowContext(ctx, "SELECT 1 FROM photos WHERE id = ? LIMIT 1", id).Scan(&found)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanPhoto(s scanner) (photo.Photo, error) {
	var (
		p          photo.Photo
		capturedAt string
	)
	if err := s.Scan(&p.ID, &p.X, &p.Y, &p.H, &p.Width, &p.Height, &capturedAt); err != nil {
		return photo.Photo{}, err
	}
	t, err := time.Parse(time.RFC3339, capturedAt)
	if err != nil {
		return photo.Photo{}, fmt.Errorf("parse captured_at %q: %w", capturedAt, err)
	}
	p.CapturedAt = t
	return p, nil
}

func (r *Repository) attachPredictions(ctx context.Context, photos []photo.Photo) error {
	if len(photos) == 0 {
		return nil
	}
	placeholders := make([]string, len(photos))
	args := make([]any, len(photos))
	index := make(map[string]int, len(photos))
	for i := range photos {
		placeholders[i] = "?"
		args[i] = photos[i].ID
		index[photos[i].ID] = i
		photos[i].Predictions = []photo.Prediction{}
	}
	query := "SELECT " + colsPreds + " FROM predictions WHERE photo_id IN (" + strings.Join(placeholders, ",") + ")"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("list predictions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var pr photo.Prediction
		if err := rows.Scan(&pr.ID, &pr.PhotoID, &pr.ClassID, &pr.Confidence,
			&pr.BBox.XMin, &pr.BBox.YMin, &pr.BBox.XMax, &pr.BBox.YMax); err != nil {
			return err
		}
		if i, ok := index[pr.PhotoID]; ok {
			photos[i].Predictions = append(photos[i].Predictions, pr)
		}
	}
	return rows.Err()
}
