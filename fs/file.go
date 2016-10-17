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
	Flush() fuse.Status
	Release()
	Fsync(flags int) (code fuse.Status)
	Truncate(size uint64) fuse.Status
	Allocate(off uint64, size uint64, mode uint32) (code fuse.Status)

	// For usage by file handle
	Read(dest []byte, off int64, ctx *fuse.Context) (fuse.ReadResult, fuse.Status)
	Write(data []byte, off int64, ctx *fuse.Context) (written uint32, code fuse.Status)
}

type DefaultFile struct {
	nodefs.File
}

func NewDefaultFile() *DefaultFile {
	return &DefaultFile{
		File: nodefs.NewDefaultFile(),
	}
}

func (f *DefaultFile) Read(dest []byte, off int64, ctx *fuse.Context) (fuse.ReadResult, fuse.Status) {
	return nil, fuse.ENOSYS
}

func (f *DefaultFile) Write(data []byte, off int64, ctx *fuse.Context) (written uint32, code fuse.Status) {
	return 0, fuse.ENOSYS
}
