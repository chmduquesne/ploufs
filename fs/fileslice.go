// copyright 2016 Christophe-Marie Duquesne

package fs

import "fmt"

type FileSlice struct {
	offset int64
	data   []byte
}

func (s *FileSlice) Beg() int {
	return int(s.offset)
}

func (s *FileSlice) End() int {
	return int(s.offset) + len(s.data)
}

func (s *FileSlice) String() string {
	return fmt.Sprintf("FileSlice{%v, %v }", s.offset, s.data)
}

func (s *FileSlice) Overlaps(other *FileSlice) bool {
	// Beginning of the other slice inside s
	if other.Beg() >= s.Beg() && other.Beg() <= s.End() {
		return true
	}
	// End of the other slice inside s
	if other.End() >= s.Beg() && other.End() <= s.End() {
		return true
	}
	// s is contained in the other slice
	if other.Beg() <= s.Beg() && other.End() >= s.End() {
		return true
	}
	return false

}

func (s *FileSlice) Merge(other *FileSlice) *FileSlice {
	// We assume the slices overlap

	min64 := func(a, b int64) int64 {
		if a < b {
			return a
		}
		return b
	}
	offset := min64(s.offset, other.offset)

	min := func(a, b int) int {
		if a < b {
			return a
		}
		return b
	}
	max := func(a, b int) int {
		if a > b {
			return a
		}
		return b
	}
	l := max(s.End(), other.End()) - min(s.Beg(), other.Beg())
	data := make([]byte, l, l)

	if s.Beg() < other.Beg() {
		copy(data, s.data)
		copy(data[other.Beg()-s.Beg():], other.data)
	} else {
		copy(data[s.Beg()-other.Beg():], s.data)
		copy(data, other.data)
	}

	res := &FileSlice{
		data:   data,
		offset: offset,
	}
	return res
}

func (c *CacheFile) Write(data []byte, off int64) {
	toInsert := &FileSlice{
		data:   data,
		offset: off,
	}
	// Create the slice that merges all mergeable slices
	for _, s := range c.slices {
		if s.Overlaps(toInsert) {
			toInsert = s.Merge(toInsert)
		}
	}

	// Keep the slice sorted by offset and non overlapping
	slices := make([]*FileSlice, 0)
	inserted := false
	for _, s := range c.slices {
		if !s.Overlaps(toInsert) {
			if s.offset > toInsert.offset && !inserted {
				slices = append(slices, toInsert)
				inserted = true
			}
			slices = append(slices, s)
		}
	}
	c.slices = slices
}
