package fs

import (
	"time"

	"github.com/hanwen/go-fuse/fuse"
)

type OverlayAttr struct {
	attr    *fuse.Attr
	deleted bool
}

func NewOverlayAttr(fs *BufferFS, path string, mode uint32, context *fuse.Context) *OverlayAttr {
	// There might be a file which we overlay as deleted. In that case we
	// don't want the same attr, because we might want to change the type.
	attr, status := fs.GetAttr(path, context)
	if status != fuse.OK {
		//rootattr, status := fs.GetAttr(path, context)
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
	b := &OverlayAttr{
		attr:    attr,
		deleted: false,
	}
	return b
}

func (n *OverlayAttr) Size() uint64 {
	return n.attr.Size
}

func (n *OverlayAttr) SetSize(sz uint64) {
	n.attr.Size = sz
}

func (n *OverlayAttr) Deleted() bool {
	return n.deleted
}

func (n *OverlayAttr) MarkDeleted() {
	n.deleted = true
}

func (n *OverlayAttr) GetAttr(out *fuse.Attr) (code fuse.Status) {
	if n.Deleted() {
		return fuse.ENOENT
	}
	out.Ino = n.attr.Ino
	out.Size = n.attr.Size
	out.Blocks = n.attr.Blocks
	out.Atime = n.attr.Atime
	out.Mtime = n.attr.Mtime
	out.Ctime = n.attr.Ctime
	out.Atimensec = n.attr.Atimensec
	out.Mtimensec = n.attr.Mtimensec
	out.Ctimensec = n.attr.Ctimensec
	out.Mode = n.attr.Mode
	out.Nlink = n.attr.Nlink
	out.Owner = n.attr.Owner
	out.Rdev = n.attr.Rdev
	out.Blksize = n.attr.Blksize
	out.Padding = n.attr.Padding
	return fuse.OK
}

func (n *OverlayAttr) Utimens(a *time.Time, m *time.Time) fuse.Status {
	n.attr.SetTimes(a, m, nil)
	return fuse.OK
}

func (n *OverlayAttr) Chmod(mode uint32) fuse.Status {
	n.attr.Mode = (n.attr.Mode & 0xfe00) | mode
	return fuse.OK
}

func (n *OverlayAttr) Chown(uid uint32, gid uint32) fuse.Status {
	n.attr.Uid = uid
	n.attr.Gid = gid
	return fuse.OK
}
