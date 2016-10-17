package fs

import "github.com/hanwen/go-fuse/fuse"

type FileHandle interface {
	Read(dest []byte, off int64) (fuse.ReadResult, fuse.Status)
	Write(data []byte, off int64) (written uint32, code fuse.Status)
}
