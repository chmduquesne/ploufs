// copyright 2016 Christophe-Marie Duquesne

package fs

import (
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
)

type attrResponse struct {
	attr   *fuse.Attr
	status fuse.Status
}

type dirResponse struct {
	entries []fuse.DirEntry
	status  fuse.Status
}

type BufferFS struct {
	pathfs.FileSystem
	bufferedWrites map[string]*BufferFile
	bufferedStats  map[string]*fuse.StatfsOut
	bufferedAttr   map[string]*attrResponse
	bufferedDir    map[string]*dirResponse
}

func NewBufferFS(wrapped pathfs.FileSystem) pathfs.FileSystem {
	return &BufferFS{
		FileSystem: wrapped,
	}
}

func (fs *BufferFS) StatFs(name string) *fuse.StatfsOut {
	cached := fs.bufferedStats[name]
	if cached != nil {
		return cached
	}
	return fs.FileSystem.StatFs(name)
}

func (fs *BufferFS) GetAttr(name string, context *fuse.Context) (a *fuse.Attr, code fuse.Status) {
	cached := fs.bufferedAttr[name]
	if cached != nil {
		return cached.attr, cached.status
	}
	return fs.FileSystem.GetAttr(name, context)
}

func (fs *BufferFS) OpenDir(name string, context *fuse.Context) (stream []fuse.DirEntry, status fuse.Status) {
	cached := fs.bufferedDir[name]
	if cached != nil {
		return cached.entries, cached.status
	}
	return fs.FileSystem.OpenDir(name, context)
}

func (fs *BufferFS) Open(name string, flags uint32, context *fuse.Context) (nodefs.File, fuse.Status) {
	fusefile, status := fs.Open(name, flags, context)
	if status != fuse.OK {
		return nil, status
	}
	return NewBufferFile(fusefile), status
}
