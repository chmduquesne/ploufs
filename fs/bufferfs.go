// copyright 2016 Christophe-Marie Duquesne

package fs

import "github.com/hanwen/go-fuse/fuse/pathfs"

type BufferFS struct {
	pathfs.FileSystem
	files map[string]BufferFile
}

func NewBufferFS(wrapped pathfs.FileSystem) pathfs.FileSystem {
	return &BufferFS{
		FileSystem: wrapped,
	}
}
