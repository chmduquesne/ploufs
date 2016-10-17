// copyright 2016 Christophe-Marie Duquesne

package fs

import (
	"fmt"
	"log"

	"github.com/hanwen/go-fuse/fuse"
)

type OverlaySymlink struct {
	File
	Dir
	Attr
	target string
}

func NewOverlaySymlink(fs *BufferFS, name string, target string, context *fuse.Context) OverlayPath {
	log.Printf("Creating overlay symlink '%s' -> '%s'\n", name, target)
	s := &OverlaySymlink{
		File:   NewDefaultFile(),
		Dir:    NewDefaultDir(),
		Attr:   NewAttr(fs, name, fuse.S_IFLNK|0777, context),
		target: target,
	}
	return s
}

func (s *OverlaySymlink) String() string {
	return fmt.Sprintf("OverlaySymlink{target: '%s'}", s.target)
}

func (s *OverlaySymlink) Target() (target string, code fuse.Status) {
	return s.target, fuse.OK
}
