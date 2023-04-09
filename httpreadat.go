// Package httpreadat provides an io.ReaderAt for http requests using the Range header.
package httpreadat

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

type RangeReader struct {
	url          string
	roundTripper http.RoundTripper
	cacheHandler CacheHandler
}

func New(url string, opts ...Option) *RangeReader {
	rr := RangeReader{
		url: url,
	}

	for _, opt := range opts {
		opt.set(&rr)
	}

	return &rr
}

func (rr *RangeReader) rawReadAt(p []byte, off int64) (n int, err error) {
	fetchSize := len(p)

	req, err := http.NewRequest("GET", rr.url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", off, off+int64(fetchSize-1)))

	resp, err := rr.client().Do(req)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	n, err = io.ReadFull(resp.Body, p)
	if err == io.ErrUnexpectedEOF {
		return n, io.EOF
	} else if err != nil {
		return n, err
	}

	return n, nil
}

func (rr *RangeReader) ReadAt(p []byte, off int64) (n int, err error) {
	rawFetcher := readerAt{
		readAt: rr.rawReadAt,
	}

	cacheHandler := rr.cacheHandler

	if cacheHandler == nil {
		cacheHandler = &nopCacheHandler{}
	}

	return cacheHandler.Get(p, off, rawFetcher)
}

func (rr *RangeReader) client() *http.Client {
	if rr.roundTripper == nil {
		return http.DefaultClient
	}
	return &http.Client{
		Transport: rr.roundTripper,
	}
}

func (rr *RangeReader) Size() (n int64, err error) {
	req, err := http.NewRequest("GET", rr.url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Range", "bytes=0-0")
	resp, err := rr.client().Do(req)
	if err != nil {
		return 0, err
	}

	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()

	rangeHeader := resp.Header.Get("Content-Range")
	rangeFields := strings.Fields(rangeHeader)
	if len(rangeFields) != 2 {
		return 0, invalidContentRangeErr
	}

	if strings.ToLower(rangeFields[0]) != "bytes" {
		return 0, invalidContentRangeErr
	}

	amts := strings.Split(rangeFields[1], "/")

	if len(amts) != 2 {
		return 0, invalidContentRangeErr
	}

	if amts[1] == "*" {
		return 0, invalidContentRangeErr
	}

	n, err = strconv.ParseInt(amts[1], 10, 64)
	if err != nil {
		return 0, invalidContentRangeErr
	}

	return n, nil
}

type nopCacheHandler struct {
}

func (h *nopCacheHandler) Get(p []byte, off int64, fetcher io.ReaderAt) (int, error) {
	return fetcher.ReadAt(p, off)
}

type readerAt struct {
	readAt func(p []byte, off int64) (n int, err error)
}

func (r readerAt) ReadAt(p []byte, off int64) (n int, err error) {
	return r.readAt(p, off)
}

var invalidContentRangeErr = errors.New("invalid Content-Range response")
