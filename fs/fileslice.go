// copyright 2016 Christophe-Marie Duquesne

package fs

import (
	"fmt"
	"log"

	"github.com/hanwen/go-fuse/fuse"
)

type FileSlice struct {
	offset int64
	data   []byte
}

// So that FileSlice satisfies the fuse.ReadResult interface
func (s *FileSlice) Done() {
}

// So that FileSlice satisfies the fuse.ReadResult interface
func (s *FileSlice) Bytes(buf []byte) ([]byte, fuse.Status) {
	return s.data, fuse.OK
}

// So that FileSlice satisfies the fuse.ReadResult interface
func (s *FileSlice) Size() int {
	return len(s.data)
}

// Returns a shorter FileSlice (beware that data is not a copy)
func (s *FileSlice) Shortened(l int) *FileSlice {
	if l > len(s.data) {
		l = len(s.data)
	}
	return &FileSlice{
		offset: s.offset,
		data:   s.data[:l],
	}
}

// Returns a shorter FileSlice (by absolute offset)
func (s *FileSlice) Truncated(off int64) *FileSlice {
	if off <= s.Beg() {
		return s.Shortened(0)
	} else {
		return s.Shortened(int(off - s.Beg()))
	}
}

func (s *FileSlice) Beg() int64 {
	return s.offset
}

func (s *FileSlice) End() int64 {
	return s.offset + int64(len(s.data))
}

func (s *FileSlice) String() string {
	data := fmt.Sprintf("%v", s.data)
	threshold := 10
	if len(s.data) > threshold {
		data = fmt.Sprintf("%v...", s.data[:threshold])
	}
	return fmt.Sprintf("FileSlice{%v, (len=%v) %v}", s.offset, len(s.data), data)
}

func (s *FileSlice) Overlaps(other *FileSlice) (res bool) {
	res = false
	// Beginning of the other slice inside s
	if other.Beg() >= s.Beg() && other.Beg() <= s.End() {
		res = true
	}
	// End of the other slice inside s
	if other.End() >= s.Beg() && other.End() <= s.End() {
		res = true
	}
	// s is contained in the other slice
	if other.Beg() <= s.Beg() && other.End() >= s.End() {
		res = true
	}
	log.Printf("[%v, %v] overlaps [%v, %v]? -> %v", s.Beg(), s.End(), other.Beg(), other.End(), res)
	return

}

func (s *FileSlice) BringIn(other *FileSlice) *FileSlice {
	// We assume the slices overlap

	min := func(a, b int64) int64 {
		if a < b {
			return a
		}
		return b
	}
	max := func(a, b int64) int64 {
		if a > b {
			return a
		}
		return b
	}

	offset := min(s.offset, other.offset)

	l := max(s.End(), other.End()) - min(s.Beg(), other.Beg())
	data := make([]byte, int(l), int(l))

	if s.Beg() < other.Beg() {
		copy(data, s.data)
		copy(data[other.Beg()-s.Beg():], other.data)
	} else {
		copy(data[s.Beg()-other.Beg():], s.data)
		copy(data, other.data)
	}

	return &FileSlice{
		data:   data,
		offset: offset,
	}
}
