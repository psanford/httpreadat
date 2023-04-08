package httpreadat

import "io"

// CacheHandler is the interface used for optional response caching.
type CacheHandler interface {
	// Get receives the original p and off passed to ReadAt.
	// If the data is not available Get can call `fetcher.ReadAt`
	// to make an http request. Get is allowed to make requests
	// that are different from the original and can invoke fetcher
	// multiple times.
	Get(p []byte, off int64, fetcher io.ReaderAt) (int, error)
}
