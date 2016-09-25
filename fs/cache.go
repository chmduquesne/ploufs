// copyright 2016 Christophe-Marie Duquesne

package fs

import (
	"syscall"
)

type FileSlice struct {
	offset int64
	data   []byte
}

type CacheFile struct {
	Statfs  syscall.Statfs_t
	deleted bool
	data    []FileSlice
}

type Cache struct {
	Files map[string]CacheFile
}
