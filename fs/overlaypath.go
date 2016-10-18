package fs

import (
	"time"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
)

type OverlayPath interface {
	// Methods of nodefs.File
	SetInode(*nodefs.Inode)
	String() string
	InnerFile() nodefs.File
	Flush() fuse.Status
	Release()
	Fsync(flags int) (code fuse.Status)
	Truncate(size uint64) fuse.Status
	GetAttr(out *fuse.Attr) fuse.Status
	Chown(uid uint32, gid uint32) fuse.Status
	Chmod(perms uint32) fuse.Status
	Utimens(atime *time.Time, mtime *time.Time) fuse.Status
	Allocate(off uint64, size uint64, mode uint32) (code fuse.Status)

	// Methods from Dir
	Entries(*fuse.Context) (stream []fuse.DirEntry, code fuse.Status)
	AddEntry(mode uint32, name string) (code fuse.Status)
	RemoveEntry(name string) (code fuse.Status)

	// Methods from symlink
	Target() (target string, code fuse.Status)

	// Methods from filehandle
	Read(dest []byte, off int64, ctx *fuse.Context, fs pathfs.FileSystem) (fuse.ReadResult, fuse.Status)
	Write(data []byte, off int64, ctx *fuse.Context, fs pathfs.FileSystem) (written uint32, code fuse.Status)
}
