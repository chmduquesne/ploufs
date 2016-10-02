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
		FileSystem:     wrapped,
		bufferedWrites: make(map[string]*BufferFile),
		bufferedStats:  make(map[string]*fuse.StatfsOut),
		bufferedAttr:   make(map[string]*attrResponse),
		bufferedDir:    make(map[string]*dirResponse),
	}
}

func (fs *BufferFS) StatFs(name string) *fuse.StatfsOut {
	cached := fs.bufferedStats[name]
	if cached != nil {
		return cached
	}
	return fs.FileSystem.StatFs(name)
}

func (fs *BufferFS) OnMount(nodeFs *pathfs.PathNodeFs) {}

func (fs *BufferFS) OnUnmount() {}

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
	res := fs.bufferedWrites[name]
	if res == nil {
		wrappedfile, status := fs.FileSystem.Open(name, flags, context)
		if status != fuse.OK {
			return nil, status
		}
		res = NewBufferFile(wrappedfile)
		fs.bufferedWrites[name] = res
	}
	return res, fuse.OK
}

func (fs *BufferFS) Chmod(path string, mode uint32, context *fuse.Context) (code fuse.Status) {
	return fuse.OK
}

func (fs *BufferFS) Chown(path string, uid uint32, gid uint32, context *fuse.Context) (code fuse.Status) {
	return fuse.OK
}

func (fs *BufferFS) Truncate(path string, offset uint64, context *fuse.Context) (code fuse.Status) {
	return fuse.OK
}

func (fs *BufferFS) Readlink(name string, context *fuse.Context) (out string, code fuse.Status) {
	return "link", fuse.OK
}

func (fs *BufferFS) Mknod(name string, mode uint32, dev uint32, context *fuse.Context) (code fuse.Status) {
	return fuse.OK
}

func (fs *BufferFS) Mkdir(path string, mode uint32, context *fuse.Context) (code fuse.Status) {
	return fuse.OK
}

// Don't use os.Remove, it removes twice (unlink followed by rmdir).
func (fs *BufferFS) Unlink(name string, context *fuse.Context) (code fuse.Status) {
	return fuse.OK
}

func (fs *BufferFS) Rmdir(name string, context *fuse.Context) (code fuse.Status) {
	return fuse.OK
}

func (fs *BufferFS) Symlink(pointedTo string, linkName string, context *fuse.Context) (code fuse.Status) {
	return fuse.OK
}

func (fs *BufferFS) Rename(oldPath string, newPath string, context *fuse.Context) (codee fuse.Status) {
	return fuse.OK
}

func (fs *BufferFS) Link(orig string, newName string, context *fuse.Context) (code fuse.Status) {
	return fuse.OK
}

func (fs *BufferFS) Access(name string, mode uint32, context *fuse.Context) (code fuse.Status) {
	return fuse.OK
}

func (fs *BufferFS) Create(name string, flags uint32, mode uint32, context *fuse.Context) (fuseFile nodefs.File, code fuse.Status) {
	res := fs.bufferedWrites[name]
	if res == nil {
		wrappedfile, status := fs.FileSystem.Create(name, flags, mode, context)
		if status != fuse.OK {
			return nil, status
		}
		res = NewBufferFile(wrappedfile)
		fs.bufferedWrites[name] = res
	}
	return res, fuse.OK
}
