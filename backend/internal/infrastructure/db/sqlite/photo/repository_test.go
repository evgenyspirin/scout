package sqlitephoto

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"scout/internal/application/photoapp"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+t.TempDir()+"/test.db")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	schema := `
	CREATE TABLE photos (id TEXT PRIMARY KEY, x REAL, y REAL, h REAL, width INTEGER, height INTEGER, captured_at TEXT);
	CREATE TABLE predictions (id TEXT PRIMARY KEY, photo_id TEXT, class_id TEXT, confidence REAL,
		bbox_xmin REAL, bbox_ymin REAL, bbox_xmax REAL, bbox_ymax REAL);`
	_, err = db.Exec(schema)
	require.NoError(t, err)

	photos := []struct{ id string }{{"p1"}, {"p2"}, {"p3"}}
	for _, p := range photos {
		_, err = db.Exec(`INSERT INTO photos VALUES (?,?,?,?,?,?,?)`,
			p.id, 10.0, 20.0, 2.0, 2560, 1440, "2026-05-27T10:00:00Z")
		require.NoError(t, err)
	}
	preds := []struct {
		id, photoID, class string
		conf               float64
	}{
		{"pr1", "p1", "thrips", 0.9},
		{"pr2", "p1", "mirid", 0.4},
		{"pr3", "p2", "thrips", 0.5},
		{"pr4", "p3", "powdery_mildew", 0.8},
	}
	for _, pr := range preds {
		_, err = db.Exec(`INSERT INTO predictions VALUES (?,?,?,?,?,?,?,?)`,
			pr.id, pr.photoID, pr.class, pr.conf, 0.1, 0.1, 0.2, 0.2)
		require.NoError(t, err)
	}
	return db
}

func confPtr(v float64) *float64 { return &v }

func TestList_Filters(t *testing.T) {
	repo := NewRepository(newTestDB(t))
	ctx := context.Background()

	tests := []struct {
		name        string
		filter      photoapp.Filter
		wantIDs     []string
		wantPredFor map[string]int // photo id -> number of predictions expected (ALL of them)
	}{
		{
			name:    "no filter returns all",
			filter:  photoapp.Filter{},
			wantIDs: []string{"p1", "p2", "p3"},
		},
		{
			name:        "classId thrips",
			filter:      photoapp.Filter{ClassID: "thrips"},
			wantIDs:     []string{"p1", "p2"},
			wantPredFor: map[string]int{"p1": 2, "p2": 1},
		},
		{
			name:        "classId thrips AND minConfidence 0.7 (same prediction)",
			filter:      photoapp.Filter{ClassID: "thrips", MinConfidence: confPtr(0.7)},
			wantIDs:     []string{"p1"},
			wantPredFor: map[string]int{"p1": 2}, // matched photo still returns ALL predictions
		},
		{
			name:    "minConfidence 0.85",
			filter:  photoapp.Filter{MinConfidence: confPtr(0.85)},
			wantIDs: []string{"p1"},
		},
		{
			name:    "classId with no matches",
			filter:  photoapp.Filter{ClassID: "spider_mites"},
			wantIDs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, err := repo.List(ctx, tt.filter, "", 50)
			require.NoError(t, err)
			gotIDs := make([]string, len(page.Items))
			for i, p := range page.Items {
				gotIDs[i] = p.ID
			}
			assert.Equal(t, tt.wantIDs, gotIDs)
			for id, n := range tt.wantPredFor {
				for _, p := range page.Items {
					if p.ID == id {
						assert.Len(t, p.Predictions, n, "photo %s predictions", id)
					}
				}
			}
		})
	}
}

func TestList_CursorPagination(t *testing.T) {
	repo := NewRepository(newTestDB(t))
	ctx := context.Background()

	first, err := repo.List(ctx, photoapp.Filter{}, "", 2)
	require.NoError(t, err)
	assert.Len(t, first.Items, 2)
	assert.NotEmpty(t, first.NextToken)

	second, err := repo.List(ctx, photoapp.Filter{}, first.NextToken, 2)
	require.NoError(t, err)
	assert.Len(t, second.Items, 1)
	assert.Empty(t, second.NextToken)
}

func TestGetByID_AndExists(t *testing.T) {
	repo := NewRepository(newTestDB(t))
	ctx := context.Background()

	p, err := repo.GetByID(ctx, "p1")
	require.NoError(t, err)
	assert.Equal(t, "p1", p.ID)
	assert.Len(t, p.Predictions, 2)

	_, err = repo.GetByID(ctx, "missing")
	require.Error(t, err)
	assert.True(t, photoapp.IsNotFound(err))

	exists, err := repo.Exists(ctx, "p2")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.Exists(ctx, "missing")
	require.NoError(t, err)
	assert.False(t, exists)
}
