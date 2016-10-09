// copyright 2016 Christophe-Marie Duquesne

package fs

import (
	"path"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
)

type BufferFS struct {
	// We want a default implementation that fails for compile reasons
	pathfs.FileSystem
	// We also want a wrapped target, but we don't rely on its
	// implementation by default
	wrappedFS pathfs.FileSystem
	overlay   map[string]OverlayPath
}

func NewBufferFS(wrapped pathfs.FileSystem) pathfs.FileSystem {
	return &BufferFS{
		FileSystem: pathfs.NewDefaultFileSystem(),
		wrappedFS:  wrapped,
		overlay:    make(map[string]OverlayPath),
	}
}

//func (fs *BufferFS) StatFs(name string) *fuse.StatfsOut {
//	return fuse.ENOSYS
//}

func (fs *BufferFS) OnMount(nodeFs *pathfs.PathNodeFs) {}

func (fs *BufferFS) OnUnmount() {}

func (fs *BufferFS) GetAttr(name string, context *fuse.Context) (a *fuse.Attr, code fuse.Status) {
	overlay := fs.overlay[name]
	if overlay != nil {
		a := fuse.Attr{}
		code := overlay.GetAttr(&a)
		return &a, code
	}
	return fs.wrappedFS.GetAttr(name, context)
}

func (fs *BufferFS) OpenDir(name string, context *fuse.Context) (stream []fuse.DirEntry, status fuse.Status) {
	overlay := fs.overlay[name]
	if overlay != nil {
		if overlay.Deleted() {
			return nil, fuse.ENOENT
		} else {
			return overlay.Entries(context)
		}
	}
	return fs.wrappedFS.OpenDir(name, context)
}

func (fs *BufferFS) Open(name string, flags uint32, context *fuse.Context) (nodefs.File, fuse.Status) {
	overlay := fs.overlay[name]
	if overlay == nil {
		overlay = NewOverlayFile(fs, name, context)
		fs.overlay[name] = overlay
	}
	return overlay, fuse.OK
}

func (fs *BufferFS) Chmod(path string, mode uint32, context *fuse.Context) (code fuse.Status) {
	// TODO mark the file modified if the chmod changes something
	overlay, _ := fs.Open(path, fuse.F_OK, context)
	return overlay.Chmod(mode)
}

func (fs *BufferFS) Chown(path string, uid uint32, gid uint32, context *fuse.Context) (code fuse.Status) {
	overlay, _ := fs.Open(path, fuse.F_OK, context)
	return overlay.Chown(uid, gid)
}

func (fs *BufferFS) Truncate(path string, offset uint64, context *fuse.Context) (code fuse.Status) {
	overlay, _ := fs.Open(path, fuse.F_OK, context)
	return overlay.Truncate(offset)
}

func (fs *BufferFS) Readlink(name string, context *fuse.Context) (out string, code fuse.Status) {
	overlay := fs.overlay[name]
	if overlay != nil {
		return overlay.Target()
	}
	return fs.wrappedFS.Readlink(name, context)
}

func (fs *BufferFS) Mknod(name string, mode uint32, dev uint32, context *fuse.Context) (code fuse.Status) {
	// No point in implementing this. It is only called for creation of
	// non-directory, non-symlink, non-regular files (cf libfuse fuse.h
	// L139) and we support only those. Other cases would be character
	// special files, block special files, FIFO, and unix sockets. None of
	// those are of interest for us.
	return fuse.ENOSYS
}

func (fs *BufferFS) Mkdir(name string, mode uint32, context *fuse.Context) (code fuse.Status) {
	dirname, basename := path.Split(name)
	if dirname != "" {
		dirname = dirname[:len(dirname)-1] // remove trailing '/'
	}
	parent := NewOverlayDir(fs, dirname, 0, context)
	parent.AddEntry(fuse.S_IFDIR|mode, basename)
	fs.overlay[dirname] = parent
	child := NewOverlayDir(fs, name, mode, context)
	fs.overlay[name] = child

	return fuse.OK
}

func (fs *BufferFS) Unlink(name string, context *fuse.Context) (code fuse.Status) {
	fs.Open(name, fuse.F_OK, context)
	fs.overlay[name].MarkDeleted()
	dirname, basename := path.Split(name)
	if dirname != "" {
		dirname = dirname[:len(dirname)-1] // remove trailing '/'
	}
	parent := fs.overlay[dirname]
	if parent == nil {
		parent = NewOverlayDir(fs, dirname, 0, context)
		fs.overlay[dirname] = parent
	}
	parent.RemoveEntry(basename)
	return fuse.OK
}

//
//func (fs *BufferFS) Rmdir(name string, context *fuse.Context) (code fuse.Status) {
//	return fuse.ENOSYS
//}
//
//func (fs *BufferFS) Symlink(pointedTo string, linkName string, context *fuse.Context) (code fuse.Status) {
//	return fuse.ENOSYS
//}
//
//func (fs *BufferFS) Rename(oldPath string, newPath string, context *fuse.Context) (codee fuse.Status) {
//	return fuse.ENOSYS
//}
//
//func (fs *BufferFS) Link(orig string, newName string, context *fuse.Context) (code fuse.Status) {
//	// We don't support hard links for now
//	return fuse.ENOSYS
//}
//
//func (fs *BufferFS) Access(name string, mode uint32, context *fuse.Context) (code fuse.Status) {
//	// Everything is allowed for now
//	return fuse.OK
//}
//
//func (fs *BufferFS) Create(name string, flags uint32, mode uint32, context *fuse.Context) (fuseFile nodefs.File, code fuse.Status) {
//	res := fs.overlay[name]
//	if res == nil {
//		wrappedfile, status := fs.wrappedFS.Create(name, flags, mode, context)
//		if status != fuse.OK {
//			return nil, status
//		}
//		res = NewBufferFile(wrappedfile)
//		fs.overlay[name] = res
//	}
//	return res, fuse.OK
//}
