// copyright 2016 Christophe-Marie Duquesne

package fs

import (
	"fmt"
	"strings"
	"syscall"
)

type CacheFile struct {
	stat    *syscall.Statfs_t
	deleted bool
	slices  []*FileSlice
}

func (c *CacheFile) String() string {
	s := make([]string, 0)
	for _, slice := range c.slices {
		s = append(s, slice.String())
	}
	return fmt.Sprintf("CacheFile{%v}", strings.Join(s, ", "))
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
