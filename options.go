package httpreadat

import "net/http"

type Option interface {
	set(*RangeReader)
}

type roundTripperOption struct {
	r http.RoundTripper
}

func (o *roundTripperOption) set(rr *RangeReader) {
	rr.roundTripper = o.r
}

func WithRoundTripper(r http.RoundTripper) Option {
	return &roundTripperOption{
		r: r,
	}
}

type cacheHandlerOption struct {
	h CacheHandler
}

func (o *cacheHandlerOption) set(rr *RangeReader) {
	rr.cacheHandler = o.h
}

func WithCacheHandler(c CacheHandler) Option {
	return &cacheHandlerOption{
		h: c,
	}
}
