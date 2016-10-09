// copyright 2016 Christophe-Marie Duquesne

package fs

import (
	"fmt"
	"log"
	"time"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type OverlaySymlink struct {
	nodefs.File
	attr   *OverlayAttr
	target string
}

func NewOverlaySymlink(fs *BufferFS, name string, target string, context *fuse.Context) *OverlaySymlink {
	log.Printf("Creating overlay symlink %s -> %s\n", name, target)
	s := &OverlaySymlink{
		File:   nodefs.NewDefaultFile(),
		attr:   NewOverlayAttr(fs, name, fuse.S_IFLNK|0777, context),
		target: target,
	}
	return s
}

func (s *OverlaySymlink) AddEntry(mode uint32, name string) (code fuse.Status) {
	return fuse.ENOTDIR
}

func (s *OverlaySymlink) RemoveEntry(name string) (code fuse.Status) {
	return fuse.ENOTDIR
}

func (s *OverlaySymlink) Entries(context *fuse.Context) (stream []fuse.DirEntry, code fuse.Status) {
	return stream, fuse.ENOTDIR
}

func (s *OverlaySymlink) String() string {
	return fmt.Sprintf("OverlaySymlink{}")
}

func (s *OverlaySymlink) GetAttr(out *fuse.Attr) (code fuse.Status) {
	return s.attr.GetAttr(out)
}

func (s *OverlaySymlink) Utimens(a *time.Time, m *time.Time) fuse.Status {
	return s.attr.Utimens(a, m)
}

func (s *OverlaySymlink) Chmod(mode uint32) fuse.Status {
	return s.attr.Chmod(mode)
}

func (s *OverlaySymlink) Chown(uid uint32, gid uint32) fuse.Status {
	return s.attr.Chown(uid, gid)
}

func (s *OverlaySymlink) Target() (target string, code fuse.Status) {
	return s.target, fuse.OK
}

func (s *OverlaySymlink) SetTarget(target string) (code fuse.Status) {
	s.target = target
	return fuse.OK
}
