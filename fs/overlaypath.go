package fs

import (
	"time"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type OverlayPath interface {
	// Methods of nodefs.File
	SetInode(*nodefs.Inode)
	String() string
	InnerFile() nodefs.File
	Read(dest []byte, off int64) (fuse.ReadResult, fuse.Status)
	Write(data []byte, off int64) (written uint32, code fuse.Status)
	Flush() fuse.Status
	Release()
	Fsync(flags int) (code fuse.Status)
	Truncate(size uint64) fuse.Status
	GetAttr(out *fuse.Attr) fuse.Status
	Chown(uid uint32, gid uint32) fuse.Status
	Chmod(perms uint32) fuse.Status
	Utimens(atime *time.Time, mtime *time.Time) fuse.Status
	Allocate(off uint64, size uint64, mode uint32) (code fuse.Status)

	// To overlay a directory
	Entries(*fuse.Context) (stream []fuse.DirEntry, code fuse.Status)
	AddEntry(mode uint32, name string) (code fuse.Status)
	RemoveEntry(name string) (code fuse.Status)

	// To overlay a symlink
	Target() (target string, code fuse.Status)
	SetTarget(target string) (code fuse.Status)
}