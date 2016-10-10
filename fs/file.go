package fs

import (
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type File interface {
	// Methods of nodefs.File, from which we remove methods which overlap
	// with our own interfaces
	SetInode(*nodefs.Inode)
	String() string
	InnerFile() nodefs.File
	Read(dest []byte, off int64) (fuse.ReadResult, fuse.Status)
	Write(data []byte, off int64) (written uint32, code fuse.Status)
	Flush() fuse.Status
	Release()
	Fsync(flags int) (code fuse.Status)
	Truncate(size uint64) fuse.Status
	Allocate(off uint64, size uint64, mode uint32) (code fuse.Status)
}
