package fs

import "github.com/hanwen/go-fuse/fuse"

type Dir interface {
	Entries(*fuse.Context) (stream []fuse.DirEntry, code fuse.Status)
	AddEntry(mode uint32, name string) (code fuse.Status)
	RemoveEntry(name string) (code fuse.Status)
}

// Default implementation that fails
type DefaultDir struct{}

func NewDefaultDir() *DefaultDir {
	return &DefaultDir{}
}

func (d *DefaultDir) Entries(context *fuse.Context) (stream []fuse.DirEntry, code fuse.Status) {
	return nil, fuse.ENOTDIR
}

func (d *DefaultDir) AddEntry(mode uint32, name string) (code fuse.Status) {
	return fuse.ENOTDIR
}

func (d *DefaultDir) RemoveEntry(name string) (code fuse.Status) {
	return fuse.ENOTDIR
}
