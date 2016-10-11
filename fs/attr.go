package fs

import (
	"log"
	"time"

	"github.com/hanwen/go-fuse/fuse"
)

type Attr interface {
	GetAttr(out *fuse.Attr) fuse.Status
	Chown(uid uint32, gid uint32) fuse.Status
	Chmod(perms uint32) fuse.Status
	Utimens(atime *time.Time, mtime *time.Time) fuse.Status
	Size() uint64
	SetSize(sz uint64)
}

type DefaultAttr struct {
	attr *fuse.Attr
	path string
}

func NewAttr(fs *BufferFS, path string, mode uint32, context *fuse.Context) Attr {
	// If the file exists, gets its existing attr from GetAttr()
	attr, status := fs.GetAttr(path, context)
	if status != fuse.OK {
		fuseOwner := fuse.Owner{
			Uid: context.Uid,
			Gid: context.Gid,
		}
		a := fuse.Attr{
			Ino:       0,
			Size:      0,
			Blocks:    0,
			Atime:     0,
			Mtime:     0,
			Ctime:     0,
			Atimensec: 0,
			Mtimensec: 0,
			Ctimensec: 0,
			Mode:      mode,
			Nlink:     1,
			Owner:     fuseOwner,
			Rdev:      0,
			Blksize:   0,
			Padding:   0,
		}
		now := time.Now()
		a.SetTimes(&now, &now, &now)
		if a.IsDir() {
			a.Size = 4096
			a.Blocks = 8
		}
		attr = &a
	}
	b := &DefaultAttr{
		attr: attr,
		path: path,
	}
	return b
}

func (a *DefaultAttr) Size() uint64 {
	return a.attr.Size
}

func (a *DefaultAttr) SetSize(sz uint64) {
	a.attr.Size = sz
}

func (a *DefaultAttr) GetAttr(out *fuse.Attr) (code fuse.Status) {
	log.Printf("'%s'.GetAttr()", a.path)
	out.Ino = a.attr.Ino
	out.Size = a.attr.Size
	out.Blocks = a.attr.Blocks
	out.Atime = a.attr.Atime
	out.Mtime = a.attr.Mtime
	out.Ctime = a.attr.Ctime
	out.Atimensec = a.attr.Atimensec
	out.Mtimensec = a.attr.Mtimensec
	out.Ctimensec = a.attr.Ctimensec
	out.Mode = a.attr.Mode
	out.Nlink = a.attr.Nlink
	out.Owner = a.attr.Owner
	out.Rdev = a.attr.Rdev
	out.Blksize = a.attr.Blksize
	out.Padding = a.attr.Padding
	return fuse.OK
}

func (a *DefaultAttr) Utimens(atime *time.Time, mtime *time.Time) fuse.Status {
	a.attr.SetTimes(atime, mtime, nil)
	return fuse.OK
}

func (a *DefaultAttr) Chmod(mode uint32) fuse.Status {
	a.attr.Mode = (a.attr.Mode & 0xfe00) | mode
	return fuse.OK
}

func (a *DefaultAttr) Chown(uid uint32, gid uint32) fuse.Status {
	a.attr.Uid = uid
	a.attr.Gid = gid
	return fuse.OK
}
