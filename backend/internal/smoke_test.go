package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIngestThenRead is an end-to-end smoke test: request an upload link,
// upload one image to object storage, then read the photo metadata back and
// verify originalUrl and predictions are returned.
//
// It requires MinIO + the dataset and self-skips when MinIO is unreachable so
// that `make test` stays green on machines without the infrastructure running.
func TestIngestThenRead(t *testing.T) {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:9000"
	}
	if c, err := net.DialTimeout("tcp", endpoint, 500*time.Millisecond); err != nil {
		t.Skipf("MinIO not reachable at %s; skipping integration smoke test", endpoint)
	} else {
		_ = c.Close()
	}

	// Point the app at the dataset relative to this package directory.
	if os.Getenv("SQLITE_PATH") == "" {
		t.Setenv("SQLITE_PATH", "../../dataset/predictions.db")
	}
	t.Setenv("MINIO_BUCKET", "scout-smoke-test")

	ctx := context.Background()
	app, err := NewApp(ctx)
	require.NoError(t, err)
	defer app.Close()
	fiberApp := app.Server().App()

	// 1. Authenticate as admin.
	loginReq := httptest.NewRequest("POST", "/api/v1/auth/login",
		bytes.NewBufferString(`{"login":"admin","password":"admin123"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp, err := fiberApp.Test(loginReq, -1)
	require.NoError(t, err)
	require.Equal(t, 200, loginResp.StatusCode)
	var login struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.NewDecoder(loginResp.Body).Decode(&login))
	token := login.AccessToken

	// Find a photo that has predictions.
	listReq := httptest.NewRequest("GET", "/api/v1/photos?classId=thrips&limit=1", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listResp, err := fiberApp.Test(listReq, -1)
	require.NoError(t, err)
	require.Equal(t, 200, listResp.StatusCode)
	var page struct {
		Items []struct {
			ID          string `json:"id"`
			Predictions []any  `json:"predictions"`
		} `json:"items"`
	}
	require.NoError(t, json.NewDecoder(listResp.Body).Decode(&page))
	require.NotEmpty(t, page.Items, "dataset should contain a thrips photo")
	photoID := page.Items[0].ID

	// 2. Request a presigned upload link.
	linkReq := httptest.NewRequest("POST", "/api/v1/photos/"+photoID+"/upload-link",
		bytes.NewBufferString(`{"contentType":"image/jpeg"}`))
	linkReq.Header.Set("Authorization", "Bearer "+token)
	linkReq.Header.Set("Content-Type", "application/json")
	linkResp, err := fiberApp.Test(linkReq, -1)
	require.NoError(t, err)
	require.Equal(t, 200, linkResp.StatusCode)
	var link struct {
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
	}
	require.NoError(t, json.NewDecoder(linkResp.Body).Decode(&link))
	require.NotEmpty(t, link.URL)

	// 3. Upload a small JPEG to the presigned URL.
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			img.Set(x, y, color.RGBA{R: 0, G: 128, B: 0, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, nil))

	putReq, err := http.NewRequest(http.MethodPut, link.URL, &buf)
	require.NoError(t, err)
	for k, v := range link.Headers {
		putReq.Header.Set(k, v)
	}
	putResp, err := http.DefaultClient.Do(putReq)
	require.NoError(t, err)
	defer putResp.Body.Close()
	require.True(t, putResp.StatusCode == 200 || putResp.StatusCode == 204, "PUT status %d", putResp.StatusCode)

	// 4. Read the photo metadata and verify originalUrl + predictions.
	getReq := httptest.NewRequest("GET", "/api/v1/photos/"+photoID, nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getResp, err := fiberApp.Test(getReq, -1)
	require.NoError(t, err)
	require.Equal(t, 200, getResp.StatusCode)
	var photo struct {
		ID          string `json:"id"`
		OriginalURL string `json:"originalUrl"`
		Predictions []any  `json:"predictions"`
	}
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&photo))
	assert.Equal(t, photoID, photo.ID)
	assert.Contains(t, photo.OriginalURL, "/api/v1/photos/"+photoID+"/original")
	assert.NotEmpty(t, photo.Predictions, "matched photo must return its predictions")
}
