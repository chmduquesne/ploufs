package fs

import (
	"syscall"

	"github.com/hanwen/go-fuse/fuse"
)

type Symlink interface {
	Target() (target string, code fuse.Status)
}

// Default implementation that fails
type DefaultSymlink struct{}

func NewDefaultSymlink() *DefaultSymlink {
	return &DefaultSymlink{}
}

func (s *DefaultSymlink) Target() (target string, code fuse.Status) {
	return "", fuse.ToStatus(syscall.ENOLINK)
}
