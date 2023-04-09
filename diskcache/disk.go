package diskcache

import (
	"io"
	"os"
)

// DiskCacheHandler caches reads to an *os.File
type DiskCacheHandler struct {
	// cache page size. 0 will use the default page size.
	// This value should only be changed before the cache handler
	// is PageSize set
	PageSize int

	f        *os.File
	fileSize int64
	pages    map[int64]bool

	cacheHit  int
	cacheMiss int
}

const defaultPageSize = 1 << 16

// NewDiskCache returns a CacheHandler strategy that caches
// reads to f. fileSize should be the size in bytes of the upstream
// object that is being cached.
// The state of the cache is stored in memory so it is not safe
// to share a cache file between different readers.
func NewDiskCache(f *os.File, fileSize int64) *DiskCacheHandler {
	return &DiskCacheHandler{
		f:        f,
		fileSize: fileSize,
		pages:    make(map[int64]bool),
	}
}

func (h *DiskCacheHandler) Get(p []byte, off int64, fetcher io.ReaderAt) (int, error) {
	if h.PageSize < 1 {
		h.PageSize = defaultPageSize
	}
	startPage, endPage := h.pagesForRange(off, len(p))

	firstMissingPage := int64(-1)
	lastMissingPage := int64(-1)

	for i := int64(startPage); i <= endPage; i++ {
		if h.pages[i] {
			continue
		}
		if firstMissingPage < 0 {
			firstMissingPage = i
		}
		if lastMissingPage < i {
			lastMissingPage = i
		}
	}

	lastPage := h.fileSize / int64(h.PageSize)

	if firstMissingPage >= 0 {
		h.cacheMiss++
		pageCount := (lastMissingPage + 1) - firstMissingPage
		size := pageCount * int64(h.PageSize)
		if lastMissingPage == lastPage {
			size = size - int64(h.PageSize) + (h.fileSize % int64(h.PageSize))
		}
		buffer := make([]byte, size)
		n, readAtErr := fetcher.ReadAt(buffer, firstMissingPage*int64(h.PageSize))
		buffer = buffer[:n]
		h.f.WriteAt(buffer, int64(firstMissingPage*int64(h.PageSize)))
		fullPagesRead := n / h.PageSize
		for i := int64(0); i < int64(fullPagesRead); i++ {
			h.pages[firstMissingPage+i] = true
		}

		if readAtErr != nil {
			// since we were trying to fetch a page instead of the original request we can't
			// just write to p and return n here.
			return 0, readAtErr
		}
	} else {
		h.cacheHit++
	}

	return h.f.ReadAt(p, off)
}

func (h *DiskCacheHandler) pagesForRange(offset int64, size int) (startPage, endPage int64) {
	startPage = offset / int64(h.PageSize)
	endPage = (offset + int64(size)) / int64(h.PageSize)

	return startPage, endPage
}
