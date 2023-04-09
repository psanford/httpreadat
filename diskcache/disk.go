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
	fileSize int
	pages    map[int]bool

	cacheHit  int
	cacheMiss int
}

const defaultPageSize = 1 << 16

// NewDiskCache returns a CacheHandler strategy that caches
// reads to f. fileSize should be the size in bytes of the upstream
// object that is being cached.
// The state of the cache is stored in memory so it is not safe
// to share a cache file between different readers.
func NewDiskCache(f *os.File, fileSize int) *DiskCacheHandler {
	return &DiskCacheHandler{
		f:        f,
		fileSize: fileSize,
		pages:    make(map[int]bool),
	}
}

func (h *DiskCacheHandler) Get(p []byte, off int64, fetcher io.ReaderAt) (int, error) {
	if h.PageSize < 1 {
		h.PageSize = defaultPageSize
	}
	startPage, endPage := h.pagesForRange(off, len(p))

	firstMissingPage := -1
	lastMissingPage := -1

	for i := startPage; i <= endPage; i++ {
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

	lastPage := h.fileSize / h.PageSize

	if firstMissingPage >= 0 {
		h.cacheMiss++
		pageCount := (lastMissingPage + 1) - firstMissingPage
		size := pageCount * h.PageSize
		if lastMissingPage == lastPage {
			size = size - h.PageSize + (h.fileSize % h.PageSize)
		}
		buffer := make([]byte, size)
		n, readAtErr := fetcher.ReadAt(buffer, int64(firstMissingPage*h.PageSize))
		buffer = buffer[:n]
		h.f.WriteAt(buffer, int64(firstMissingPage*h.PageSize))
		fullPagesRead := n / h.PageSize
		for i := 0; i < fullPagesRead; i++ {
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

func (h *DiskCacheHandler) pagesForRange(offset int64, size int) (startPage, endPage int) {
	startPage = int(offset) / h.PageSize
	endPage = (int(offset) + size) / h.PageSize

	return startPage, endPage
}
