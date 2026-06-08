package rest_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"scout/internal/application/authapp"
	"scout/internal/application/photoapp"
	"scout/internal/application/thumbapp"
	"scout/internal/domain/photo"
	"scout/internal/infrastructure/jwt"
	"scout/internal/infrastructure/metrics"
	"scout/internal/infrastructure/users"
	"scout/internal/interface/api/rest"
)

// --- Fakes (no infrastructure needed) ---

type fakeRepo struct{ lastFilter photoapp.Filter }

func samplePhoto(id string, n int) photo.Photo {
	preds := make([]photo.Prediction, n)
	for i := 0; i < n; i++ {
		preds[i] = photo.Prediction{
			ClassID:    "thrips",
			Confidence: 0.8,
			BBox:       photo.BoundingBox{XMin: 0, YMin: 0, XMax: 0.5, YMax: 0.5},
		}
	}
	return photo.Photo{ID: id, X: 1, Y: 2, H: 2, Width: 2560, Height: 1440, CapturedAt: time.Now(), Predictions: preds}
}

func (f *fakeRepo) List(_ context.Context, flt photoapp.Filter, _ string, _ int) (photoapp.Page, error) {
	f.lastFilter = flt
	return photoapp.Page{Items: []photo.Photo{samplePhoto("a", 2), samplePhoto("b", 1)}}, nil
}
func (f *fakeRepo) GetByID(_ context.Context, id string) (photo.Photo, error) {
	if id == "missing" {
		return photo.Photo{}, photoapp.NotFound(id)
	}
	return samplePhoto(id, 2), nil
}
func (f *fakeRepo) Exists(_ context.Context, _ string) (bool, error) { return true, nil }

type fakeStorage struct{}

func (fakeStorage) PresignPut(_ context.Context, _, _ string, ttl time.Duration) (string, map[string]string, time.Time, error) {
	return "http://minio.local/put", map[string]string{"Content-Type": "image/jpeg"}, time.Now().Add(ttl), nil
}
func (fakeStorage) ObjectExists(_ context.Context, _ string) (bool, error) { return true, nil }

type fakeStreamer struct{}

func (fakeStreamer) StreamOriginal(_ context.Context, _ string) (io.ReadCloser, string, int64, bool, error) {
	return io.NopCloser(strings.NewReader("img")), "image/jpeg", 3, true, nil
}

type fakeThumbCache struct{}

func (fakeThumbCache) Get(_ context.Context, _ string) (thumbapp.Thumbnail, string, bool) {
	return thumbapp.Thumbnail{}, "", false
}
func (fakeThumbCache) Set(_ context.Context, _ string, _ thumbapp.Thumbnail) {}

type fakeOriginalStore struct{}

func (fakeOriginalStore) GetOriginal(_ context.Context, _ string) ([]byte, bool, error) {
	return nil, false, nil
}

type fakeGenerator struct{}

func (fakeGenerator) Generate(_ context.Context, _ []byte, _ thumbapp.Params) (thumbapp.Thumbnail, error) {
	return thumbapp.Thumbnail{}, nil
}

func newTestServer(t *testing.T) *rest.Server {
	t.Helper()
	logger := zap.NewNop()
	mtr := metrics.New()
	jwtSvc := jwt.New("test-secret")
	userRepo, err := users.NewRepository(users.DefaultCredentials())
	require.NoError(t, err)

	authSvc := authapp.NewService(userRepo, jwtSvc, time.Hour)
	photoSvc := photoapp.NewService(&fakeRepo{}, fakeStorage{}, time.Minute, "http://test")
	thumbSvc := thumbapp.NewService(fakeThumbCache{}, fakeOriginalStore{}, fakeGenerator{}, mtr)

	return rest.NewServer(rest.ServerDeps{
		Logger:     logger,
		Metrics:    mtr,
		JWT:        jwtSvc,
		Auth:       rest.NewAuthController(logger, authSvc),
		Photos:     rest.NewPhotoController(logger, photoSvc, fakeStreamer{}),
		Thumbnails: rest.NewThumbnailController(logger, thumbSvc, 3600),
		Ops:        rest.NewOpsController(mtr),
		BodyLimit:  1 << 20,
	})
}

func loginToken(t *testing.T, srv *rest.Server, login, pass string) string {
	t.Helper()
	body := strings.NewReader(`{"login":"` + login + `","password":"` + pass + `"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := srv.App().Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	var out struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	require.NotEmpty(t, out.AccessToken)
	return out.AccessToken
}

func TestListPhotos_RequiresAuth(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest("GET", "/api/v1/photos", nil)
	resp, err := srv.App().Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)
	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "AuthenticationRequired", body["code"])
	assert.NotEmpty(t, body["request_id"])
}

func TestListPhotos_ReturnsAllPredictions(t *testing.T) {
	srv := newTestServer(t)
	token := loginToken(t, srv, "insect", "insect123")
	req := httptest.NewRequest("GET", "/api/v1/photos?classId=thrips&minConfidence=0.7", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := srv.App().Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)

	var page struct {
		Items []struct {
			ID          string `json:"id"`
			OriginalURL string `json:"originalUrl"`
			Predictions []any  `json:"predictions"`
		} `json:"items"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&page))
	require.Len(t, page.Items, 2)
	// A matched photo returns ALL of its predictions.
	assert.Len(t, page.Items[0].Predictions, 2)
	assert.Contains(t, page.Items[0].OriginalURL, "/api/v1/photos/a/original")
}

func TestThumbnail_InvalidParams(t *testing.T) {
	srv := newTestServer(t)
	token := loginToken(t, srv, "insect", "insect123")
	req := httptest.NewRequest("GET", "/api/v1/photos/a/thumbnail?width=999", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := srv.App().Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "ValidationError", body["code"])
}

func TestUploadLink_AdminOnly(t *testing.T) {
	srv := newTestServer(t)
	userToken := loginToken(t, srv, "insect", "insect123")
	req := httptest.NewRequest("POST", "/api/v1/photos/a/upload-link", strings.NewReader(`{"contentType":"image/jpeg"}`))
	req.Header.Set("Authorization", "Bearer "+userToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := srv.App().Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)

	adminToken := loginToken(t, srv, "admin", "admin123")
	req2 := httptest.NewRequest("POST", "/api/v1/photos/a/upload-link", strings.NewReader(`{"contentType":"image/jpeg"}`))
	req2.Header.Set("Authorization", "Bearer "+adminToken)
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := srv.App().Test(req2, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp2.StatusCode)
}

// TestMetricsScrape guards against the Fiber pooled-buffer label-corruption bug:
// after several requests across different methods/paths, scraping /metrics as an
// admin must return 200 (not a Prometheus "collected before" 500).
func TestMetricsScrape(t *testing.T) {
	srv := newTestServer(t)
	userToken := loginToken(t, srv, "insect", "insect123")
	adminToken := loginToken(t, srv, "admin", "admin123")

	// Generate traffic across methods/paths so labels are stored in the registry.
	for i := 0; i < 5; i++ {
		for _, r := range []struct{ method, path, token string }{
			{"GET", "/api/v1/photos", userToken},
			{"GET", "/api/v1/photos/a", userToken},
			{"GET", "/api/v1/photos/missing", userToken},
			{"HEAD", "/api/v1/photos/a/object", adminToken},
		} {
			req := httptest.NewRequest(r.method, r.path, nil)
			req.Header.Set("Authorization", "Bearer "+r.token)
			resp, err := srv.App().Test(req, -1)
			require.NoError(t, err)
			_ = resp.Body.Close()
		}
	}

	req := httptest.NewRequest("GET", "/metrics", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err := srv.App().Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "http_requests_total")
}
