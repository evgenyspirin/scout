package thumbapp

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Fakes for the thumbnail engine collaborators ---

type fakeCache struct {
	mu      sync.Mutex
	getFn   func(key string) (Thumbnail, string, bool)
	getN    int32
	setKeys []string
	setVals []Thumbnail
}

func (c *fakeCache) Get(_ context.Context, key string) (Thumbnail, string, bool) {
	atomic.AddInt32(&c.getN, 1)
	if c.getFn != nil {
		return c.getFn(key)
	}
	return Thumbnail{}, "", false // default: always miss
}

func (c *fakeCache) Set(_ context.Context, key string, t Thumbnail) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setKeys = append(c.setKeys, key)
	c.setVals = append(c.setVals, t)
}

func (c *fakeCache) setCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.setKeys)
}

type fakeStore struct {
	mu      sync.Mutex
	data    []byte
	found   bool
	err     error
	n       int32
	lastKey string
}

func (s *fakeStore) GetOriginal(_ context.Context, key string) ([]byte, bool, error) {
	atomic.AddInt32(&s.n, 1)
	s.mu.Lock()
	s.lastKey = key
	s.mu.Unlock()
	return s.data, s.found, s.err
}

type fakeGen struct {
	out  Thumbnail
	err  error
	n    int32
	hook func() // optional: used to control timing in concurrency tests
}

func (g *fakeGen) Generate(_ context.Context, _ []byte, _ Params) (Thumbnail, error) {
	atomic.AddInt32(&g.n, 1)
	if g.hook != nil {
		g.hook()
	}
	return g.out, g.err
}

type fakeMetrics struct {
	mu          sync.Mutex
	requested   int
	miss        int
	hits        map[string]int
	generated   int
	genErr      int
	started     int
	finished    int
	lastSeconds float64
}

func newFakeMetrics() *fakeMetrics { return &fakeMetrics{hits: map[string]int{}} }

func (m *fakeMetrics) ThumbnailRequested()  { m.mu.Lock(); m.requested++; m.mu.Unlock() }
func (m *fakeMetrics) ThumbnailCacheMiss()  { m.mu.Lock(); m.miss++; m.mu.Unlock() }
func (m *fakeMetrics) GenerationStarted()   { m.mu.Lock(); m.started++; m.mu.Unlock() }
func (m *fakeMetrics) GenerationFinished()  { m.mu.Lock(); m.finished++; m.mu.Unlock() }
func (m *fakeMetrics) ThumbnailGenerationError() {
	m.mu.Lock()
	m.genErr++
	m.mu.Unlock()
}
func (m *fakeMetrics) ThumbnailCacheHit(level string) {
	m.mu.Lock()
	m.hits[level]++
	m.mu.Unlock()
}
func (m *fakeMetrics) ThumbnailGenerated(seconds float64) {
	m.mu.Lock()
	m.generated++
	m.lastSeconds = seconds
	m.mu.Unlock()
}

func (m *fakeMetrics) snapshot() fakeMetrics {
	m.mu.Lock()
	defer m.mu.Unlock()
	hits := make(map[string]int, len(m.hits))
	for k, v := range m.hits {
		hits[k] = v
	}
	return fakeMetrics{
		requested: m.requested, miss: m.miss, hits: hits,
		generated: m.generated, genErr: m.genErr,
		started: m.started, finished: m.finished, lastSeconds: m.lastSeconds,
	}
}

const testPhotoID = "photo-42"

var testParams = Params{Width: 640, DPR: 2, Quality: 80, Format: "jpeg"}

// --- Cache hit: must return cached bytes without touching store or generator ---

func TestGet_CacheHit_ServesWithoutGenerating(t *testing.T) {
	for _, level := range []string{"lru", "redis"} {
		t.Run(level, func(t *testing.T) {
			cached := Thumbnail{Data: []byte("cached-bytes"), ContentType: "image/jpeg"}
			cache := &fakeCache{getFn: func(string) (Thumbnail, string, bool) {
				return cached, level, true
			}}
			store := &fakeStore{}
			gen := &fakeGen{}
			m := newFakeMetrics()
			svc := NewService(cache, store, gen, m)

			got, err := svc.Get(context.Background(), testPhotoID, testParams)
			require.NoError(t, err)
			assert.Equal(t, cached, got)

			// Neither the store nor the generator should be consulted on a hit.
			assert.Equal(t, int32(0), atomic.LoadInt32(&store.n), "store must not be read on cache hit")
			assert.Equal(t, int32(0), atomic.LoadInt32(&gen.n), "generator must not run on cache hit")
			assert.Equal(t, 0, cache.setCount(), "cache must not be re-written on a hit")

			s := m.snapshot()
			assert.Equal(t, 1, s.requested)
			assert.Equal(t, 1, s.hits[level])
			assert.Equal(t, 0, s.miss)
			assert.Equal(t, 0, s.generated)
			assert.Equal(t, 0, s.started)
		})
	}
}

// --- Cache miss → fetch original → generate → store in cache ---

func TestGet_CacheMiss_GeneratesAndCaches(t *testing.T) {
	produced := Thumbnail{Data: []byte("fresh-thumb"), ContentType: "image/jpeg"}
	cache := &fakeCache{} // always miss
	store := &fakeStore{data: []byte("original-jpeg"), found: true}
	gen := &fakeGen{out: produced}
	m := newFakeMetrics()
	svc := NewService(cache, store, gen, m)

	got, err := svc.Get(context.Background(), testPhotoID, testParams)
	require.NoError(t, err)
	assert.Equal(t, produced, got)

	// Original is fetched by object key "<id>.jpg".
	assert.Equal(t, int32(1), atomic.LoadInt32(&store.n))
	store.mu.Lock()
	assert.Equal(t, testPhotoID+".jpg", store.lastKey)
	store.mu.Unlock()

	// Generated thumbnail is written back under the deterministic cache key.
	assert.Equal(t, int32(1), atomic.LoadInt32(&gen.n))
	require.Equal(t, 1, cache.setCount())
	assert.Equal(t, testParams.CacheKey(testPhotoID), cache.setKeys[0])
	assert.Equal(t, produced, cache.setVals[0])

	s := m.snapshot()
	assert.Equal(t, 1, s.requested)
	assert.Equal(t, 1, s.miss)
	assert.Equal(t, 1, s.generated)
	assert.Equal(t, 1, s.started)
	assert.Equal(t, 1, s.finished, "GenerationFinished must run (deferred)")
	assert.Equal(t, 0, s.genErr)
	assert.Empty(t, s.hits)
	assert.GreaterOrEqual(t, s.lastSeconds, 0.0, "generation duration must be observed")
}

// --- Original not uploaded yet → typed not-found error, no generation ---

func TestGet_OriginalNotFound(t *testing.T) {
	cache := &fakeCache{}
	store := &fakeStore{found: false} // object missing
	gen := &fakeGen{}
	m := newFakeMetrics()
	svc := NewService(cache, store, gen, m)

	_, err := svc.Get(context.Background(), testPhotoID, testParams)
	require.Error(t, err)
	assert.True(t, IsOriginalNotFound(err), "must be the typed original-not-found error")

	assert.Equal(t, int32(0), atomic.LoadInt32(&gen.n), "generator must not run when original is missing")
	assert.Equal(t, 0, cache.setCount(), "nothing should be cached when original is missing")

	s := m.snapshot()
	assert.Equal(t, 1, s.requested)
	assert.Equal(t, 1, s.miss)
	assert.Equal(t, 0, s.started, "generation must not start when original is missing")
	assert.Equal(t, 0, s.generated)
	assert.Equal(t, 0, s.genErr)
}

// --- Object store failure is propagated, not swallowed ---

func TestGet_StoreError_Propagates(t *testing.T) {
	boom := errors.New("storage unavailable")
	cache := &fakeCache{}
	store := &fakeStore{err: boom}
	gen := &fakeGen{}
	m := newFakeMetrics()
	svc := NewService(cache, store, gen, m)

	_, err := svc.Get(context.Background(), testPhotoID, testParams)
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
	assert.False(t, IsOriginalNotFound(err), "a real store error is not an original-not-found")

	assert.Equal(t, int32(0), atomic.LoadInt32(&gen.n))
	assert.Equal(t, 0, cache.setCount())

	s := m.snapshot()
	assert.Equal(t, 0, s.started)
	assert.Equal(t, 0, s.generated)
	assert.Equal(t, 0, s.genErr)
}

// --- Generator failure: error propagated, error metric recorded, nothing cached ---

func TestGet_GeneratorError(t *testing.T) {
	boom := errors.New("resize failed")
	cache := &fakeCache{}
	store := &fakeStore{data: []byte("original"), found: true}
	gen := &fakeGen{err: boom}
	m := newFakeMetrics()
	svc := NewService(cache, store, gen, m)

	_, err := svc.Get(context.Background(), testPhotoID, testParams)
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)

	assert.Equal(t, int32(1), atomic.LoadInt32(&gen.n))
	assert.Equal(t, 0, cache.setCount(), "failed generation must not be cached")

	s := m.snapshot()
	assert.Equal(t, 1, s.started)
	assert.Equal(t, 1, s.finished, "GenerationFinished must still run on the error path (deferred)")
	assert.Equal(t, 1, s.genErr)
	assert.Equal(t, 0, s.generated, "a failed generation must not count as generated")
}

// --- Singleflight: many identical concurrent requests trigger exactly one generation ---

func TestGet_Singleflight_CoalescesIdenticalRequests(t *testing.T) {
	const n = 25
	produced := Thumbnail{Data: []byte("one-and-only"), ContentType: "image/jpeg"}

	cache := &fakeCache{} // always miss, so every caller would otherwise generate
	store := &fakeStore{data: []byte("original"), found: true}
	// Hold the in-flight leader long enough for all followers to attach to the
	// same singleflight key before it completes.
	gen := &fakeGen{out: produced, hook: func() { time.Sleep(80 * time.Millisecond) }}
	m := newFakeMetrics()
	svc := NewService(cache, store, gen, m)

	start := make(chan struct{})
	results := make([]Thumbnail, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			<-start // release all goroutines simultaneously
			results[idx], errs[idx] = svc.Get(context.Background(), testPhotoID, testParams)
		}(i)
	}
	close(start)
	wg.Wait()

	// Exactly one generation and one original fetch despite n concurrent callers.
	assert.Equal(t, int32(1), atomic.LoadInt32(&gen.n), "generator must run exactly once")
	assert.Equal(t, int32(1), atomic.LoadInt32(&store.n), "original fetched exactly once")
	assert.Equal(t, 1, cache.setCount(), "thumbnail cached exactly once")

	// Every caller receives the same successful result.
	for i := 0; i < n; i++ {
		require.NoErrorf(t, errs[i], "caller %d", i)
		assert.Equalf(t, produced, results[i], "caller %d", i)
	}

	s := m.snapshot()
	assert.Equal(t, n, s.requested, "every caller is counted as a request")
	assert.Equal(t, 1, s.generated, "only one generation observed")
	assert.Equal(t, 1, s.started)
}

// --- Singleflight is keyed: distinct photos each generate independently ---

func TestGet_Singleflight_DistinctKeysGenerateSeparately(t *testing.T) {
	cache := &fakeCache{}
	store := &fakeStore{data: []byte("original"), found: true}
	gen := &fakeGen{out: Thumbnail{Data: []byte("t")}, hook: func() { time.Sleep(40 * time.Millisecond) }}
	svc := NewService(cache, store, gen, newFakeMetrics())

	start := make(chan struct{})
	var wg sync.WaitGroup
	ids := []string{"a", "b", "c"}
	wg.Add(len(ids))
	for _, id := range ids {
		go func(id string) {
			defer wg.Done()
			<-start
			_, _ = svc.Get(context.Background(), id, testParams)
		}(id)
	}
	close(start)
	wg.Wait()

	assert.Equal(t, int32(len(ids)), atomic.LoadInt32(&gen.n),
		"each distinct photo id must generate its own thumbnail")
}

// --- Supplemental parameter coverage: full allowed-value matrix is accepted ---

func TestParseParams_AllAllowedValuesAccepted(t *testing.T) {
	for _, w := range []int{320, 640, 960, 1280} {
		for _, d := range []int{1, 2, 3} {
			for _, q := range []int{70, 80, 90} {
				got, err := ParseParams(itoa(w), itoa(d), itoa(q), "jpeg")
				require.NoError(t, err)
				assert.Equal(t, Params{Width: w, DPR: d, Quality: q, Format: "jpeg"}, got)
			}
		}
	}
}

func TestParseParams_RejectsOutOfRangeAndJunk(t *testing.T) {
	cases := []struct {
		name, w, d, q, f, field string
	}{
		{"zero width", "0", "", "", "", "width"},
		{"negative width", "-320", "", "", "", "width"},
		{"unlisted width 500", "500", "", "", "", "width"},
		{"zero dpr", "", "0", "", "", "dpr"},
		{"dpr too high", "", "5", "", "", "dpr"},
		{"quality too low", "", "", "50", "", "quality"},
		{"quality 100", "", "", "100", "", "quality"},
		{"empty format string is default", "", "", "", "JPEG", "format"}, // case-sensitive: "JPEG" != "jpeg"
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseParams(c.w, c.d, c.q, c.f)
			require.Error(t, err)
			pe, ok := err.(ParamError)
			require.True(t, ok, "expected ParamError, got %T", err)
			assert.Equal(t, c.field, pe.Field)
		})
	}
}

// itoa is a tiny local helper to keep the matrix test readable.
func itoa(v int) string {
	if v < 0 {
		return "-" + itoa(-v)
	}
	if v < 10 {
		return string(rune('0' + v))
	}
	return itoa(v/10) + itoa(v%10)
}
