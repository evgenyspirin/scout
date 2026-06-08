package thumbapp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseParams(t *testing.T) {
	tests := []struct {
		name     string
		width    string
		dpr      string
		quality  string
		format   string
		wantErr  bool
		errField string
		want     Params
	}{
		{name: "defaults when empty", want: Params{Width: 320, DPR: 1, Quality: 80, Format: "jpeg"}},
		{name: "all valid", width: "1280", dpr: "3", quality: "90", format: "jpeg",
			want: Params{Width: 1280, DPR: 3, Quality: 90, Format: "jpeg"}},
		{name: "valid 640/2/70", width: "640", dpr: "2", quality: "70", format: "jpeg",
			want: Params{Width: 640, DPR: 2, Quality: 70, Format: "jpeg"}},
		{name: "bad width", width: "999", wantErr: true, errField: "width"},
		{name: "non-numeric width", width: "abc", wantErr: true, errField: "width"},
		{name: "bad dpr", dpr: "4", wantErr: true, errField: "dpr"},
		{name: "bad quality", quality: "100", wantErr: true, errField: "quality"},
		{name: "unsupported format webp", format: "webp", wantErr: true, errField: "format"},
		{name: "unknown format", format: "gif", wantErr: true, errField: "format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseParams(tt.width, tt.dpr, tt.quality, tt.format)
			if tt.wantErr {
				require.Error(t, err)
				pe, ok := err.(ParamError)
				require.True(t, ok, "expected ParamError")
				assert.Equal(t, tt.errField, pe.Field)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEffectiveWidth(t *testing.T) {
	tests := []struct {
		name     string
		params   Params
		original int
		want     int
	}{
		{name: "within original", params: Params{Width: 640, DPR: 2}, original: 2560, want: 1280},
		{name: "capped by original (no upscale)", params: Params{Width: 1280, DPR: 3}, original: 2560, want: 2560},
		{name: "exact match", params: Params{Width: 1280, DPR: 2}, original: 2560, want: 2560},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.params.EffectiveWidth(tt.original))
		})
	}
}

func TestCacheKeyDeterministic(t *testing.T) {
	p := Params{Width: 640, DPR: 2, Quality: 80, Format: "jpeg"}
	key := p.CacheKey("abc-123")
	assert.Equal(t, "thumbnail:abc-123:w640:dpr2:q80:fmtjpeg", key)
	// Same params always produce the same key.
	assert.Equal(t, key, p.CacheKey("abc-123"))
	// Different params produce a different key.
	other := Params{Width: 320, DPR: 1, Quality: 90, Format: "jpeg"}
	assert.NotEqual(t, key, other.CacheKey("abc-123"))
}

func TestContentType(t *testing.T) {
	assert.Equal(t, "image/jpeg", Params{Format: "jpeg"}.ContentType())
	assert.Equal(t, "image/webp", Params{Format: "webp"}.ContentType())
}
