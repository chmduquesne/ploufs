// copyright 2016 Christophe-Marie Duquesne

package fs

import (
	"fmt"

	"github.com/hanwen/go-fuse/fuse"
)

type OverlaySymlink struct {
	File
	Dir
	OverlayAttr
	target string
}

func NewOverlaySymlink(attr OverlayAttr, target string) OverlayPath {
	return &OverlaySymlink{
		File:        NewDefaultFile(),
		Dir:         NewDefaultDir(),
		OverlayAttr: attr,
		target:      target,
	}
}

func (s *OverlaySymlink) String() string {
	return fmt.Sprintf("OverlaySymlink{target: '%s'}", s.target)
}

func (s *OverlaySymlink) Target() (target string, code fuse.Status) {
	return s.target, fuse.OK
}
