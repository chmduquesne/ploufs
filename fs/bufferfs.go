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
	// First we check if the file appears in the listing of its parent
	// directory
	dirname, basename := path.Split(name)
	entries, status := fs.OpenDir(dirname, context)
	if status != fuse.OK {
		return a, status
	} else {
		found := false
		for _, e := range entries {
			if e.Name == basename {
				found = true
				break
			}
		}
		if !found {
			return a, fuse.ENOENT
		}
	}
	// At this point we know the file is listed in its parent
	// We check if we have overriden its properties somehow
	overlay := fs.overlay[name]
	if overlay != nil {
		a := fuse.Attr{}
		code := overlay.GetAttr(&a)
		return &a, code
	}
	// We did not find the file, we resort to the underlying file system
	return fs.wrappedFS.GetAttr(name, context)
}

func (fs *BufferFS) OpenDir(name string, context *fuse.Context) (stream []fuse.DirEntry, status fuse.Status) {
	overlay := fs.overlay[name]
	if overlay != nil {
		return overlay.Entries(context)
	}
	return fs.wrappedFS.OpenDir(name, context)
}

func (fs *BufferFS) Open(name string, flags uint32, context *fuse.Context) (nodefs.File, fuse.Status) {
	overlay := fs.overlay[name]
	if overlay == nil {
		overlay = NewOverlayFile(fs, name, 0, 0, context)
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

func (fs *BufferFS) Unlink(name string, context *fuse.Context) (code fuse.Status) {
	delete(fs.overlay, name)
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

func (fs *BufferFS) Rmdir(name string, context *fuse.Context) (code fuse.Status) {
	delete(fs.overlay, name)
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

func (fs *BufferFS) Symlink(target string, name string, context *fuse.Context) (code fuse.Status) {
	dirname, basename := path.Split(name)
	if dirname != "" {
		dirname = dirname[:len(dirname)-1] // remove trailing '/'
	}
	parent := fs.overlay[dirname]
	if parent == nil {
		parent = NewOverlayDir(fs, dirname, 0, context)
		fs.overlay[dirname] = parent
	}
	parent.AddEntry(fuse.S_IFLNK|0777, basename)

	child := NewOverlaySymlink(fs, name, target, context)
	fs.overlay[name] = child

	return fuse.OK
}

func (fs *BufferFS) Mkdir(name string, mode uint32, context *fuse.Context) (code fuse.Status) {
	dirname, basename := path.Split(name)
	if dirname != "" {
		dirname = dirname[:len(dirname)-1] // remove trailing '/'
	}
	parent := fs.overlay[dirname]
	if parent == nil {
		parent = NewOverlayDir(fs, dirname, 0, context)
		fs.overlay[dirname] = parent
	}
	parent.AddEntry(fuse.S_IFDIR|mode, basename)

	child := NewOverlayDir(fs, name, mode, context)
	fs.overlay[name] = child

	return fuse.OK
}

func (fs *BufferFS) Create(name string, flags uint32, mode uint32, context *fuse.Context) (fuseFile nodefs.File, code fuse.Status) {
	dirname, basename := path.Split(name)
	if dirname != "" {
		dirname = dirname[:len(dirname)-1] // remove trailing '/'
	}
	parent := fs.overlay[dirname]
	if parent == nil {
		parent = NewOverlayDir(fs, dirname, 0, context)
		fs.overlay[dirname] = parent
	}
	parent.AddEntry(fuse.S_IFREG|mode, basename)
	fs.overlay[dirname] = parent

	child := NewOverlayFile(fs, name, flags, mode, context)
	fs.overlay[name] = child

	return child, fuse.OK
}

func (fs *BufferFS) Rename(oldPath string, newPath string, context *fuse.Context) (code fuse.Status) {
	attr, status := fs.GetAttr(oldPath, context)
	if status != fuse.OK {
		return status
	}
	overlay := fs.overlay[oldPath]
	if overlay == nil {
		if attr.IsDir() {
			overlay = NewOverlayDir(fs, oldPath, attr.Mode, context)
		}
		if attr.IsRegular() {
			overlay = NewOverlayFile(fs, oldPath, 0, attr.Mode, context)
		}
		if attr.IsSymlink() {
			target, st := fs.Readlink(oldPath, context)
			if st != fuse.OK {
				return st
			}
			overlay = NewOverlaySymlink(fs, oldPath, target, context)
		}
	}

	olddirname, oldbasename := path.Split(oldPath)
	if olddirname != "" {
		olddirname = olddirname[:len(olddirname)-1] // remove trailing '/'
	}
	oldparent := fs.overlay[olddirname]
	if oldparent == nil {
		oldparent = NewOverlayDir(fs, olddirname, 0, context)
		fs.overlay[olddirname] = oldparent
	}
	oldparent.RemoveEntry(oldbasename)

	newdirname, newbasename := path.Split(oldPath)
	if newdirname != "" {
		newdirname = newdirname[:len(newdirname)-1] // remove trailing '/'
	}
	newparent := fs.overlay[newdirname]
	if newparent == nil {
		newparent = NewOverlayDir(fs, newdirname, 0, context)
		fs.overlay[newdirname] = newparent
	}
	newparent.AddEntry(attr.Mode, newbasename)

	delete(fs.overlay, oldPath)
	fs.overlay[newPath] = overlay
	return fuse.OK
}

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
