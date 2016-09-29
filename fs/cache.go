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

type Cache struct {
	files map[string]CacheFile
}
