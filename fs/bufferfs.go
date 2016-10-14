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

func pathSplit(name string) (dirname string, basename string) {
	dirname, basename = path.Split(name)
	if dirname != "" {
		dirname = dirname[:len(dirname)-1] // remove trailing '/' if present
	}
	return
}

func NewBufferFS(wrapped pathfs.FileSystem) pathfs.FileSystem {
	return &BufferFS{
		FileSystem: pathfs.NewDefaultFileSystem(),
		wrappedFS:  wrapped,
		overlay:    make(map[string]OverlayPath),
	}
}

func (fs *BufferFS) StatFs(name string) *fuse.StatfsOut {
	// We rely entirely on the underlying FS
	return fs.wrappedFS.StatFs(name)
}

func (fs *BufferFS) OnMount(nodeFs *pathfs.PathNodeFs) {}

func (fs *BufferFS) OnUnmount() {}

func (fs *BufferFS) GetAttr(name string, context *fuse.Context) (a *fuse.Attr, code fuse.Status) {
	if name != "" {
		// If a file is not listed in its parent directory, it does not exist
		// (except for the root directory which does not list itself)
		dirname, basename := pathSplit(name)
		entries, status := fs.OpenDir(dirname, context)
		if status != fuse.OK {
			// We could not open the parent
			return nil, status
		} else {
			found := false
			for _, e := range entries {
				if e.Name == basename {
					found = true
					break
				}
			}
			if !found {
				// the parent did not list us
				return nil, fuse.ENOENT
			}
		}
	}
	// The file exists, but we may have overlayed its attributes
	overlayPath := fs.overlay[name]
	if overlayPath != nil {
		a = &fuse.Attr{}
		code = overlayPath.GetAttr(a)
		return
	}
	// The file is not overlayed, we resort to the underlying file system
	a, code = fs.wrappedFS.GetAttr(name, context)
	return
}

func (fs *BufferFS) OpenDir(name string, context *fuse.Context) (stream []fuse.DirEntry, status fuse.Status) {
	overlayPath := fs.overlay[name]
	if overlayPath != nil {
		return overlayPath.Entries(context)
	}
	return fs.wrappedFS.OpenDir(name, context)
}

func (fs *BufferFS) Open(name string, flags uint32, context *fuse.Context) (nodefs.File, fuse.Status) {
	// Assumes that fuse has checked the permissions
	overlayPath := fs.overlay[name]
	if overlayPath == nil {
		overlayPath = NewOverlayFile(fs, name, flags, 0, context)
		fs.overlay[name] = overlayPath
	}
	return overlayPath, fuse.OK
}

func (fs *BufferFS) Chmod(name string, mode uint32, context *fuse.Context) (code fuse.Status) {
	// Do we need to do anything? Check the existing mode
	attr, status := fs.GetAttr(name, context)
	if status != fuse.OK {
		return status
	}
	if attr.Mode&0777 == mode {
		return fuse.OK
	}
	// The mode will change, we need to overlay
	overlayPath := fs.overlay[name]
	if overlayPath == nil {
		if attr.IsDir() {
			overlayPath = NewOverlayDir(fs, name, mode, context)
		}
		if attr.IsRegular() {
			overlayPath = NewOverlayFile(fs, name, 0, mode, context)
		}
		// Permissions on symlinks don't make sense (I think) -> TESTME
		if attr.IsSymlink() {
			return fuse.OK
		}
		fs.overlay[name] = overlayPath
	}
	return overlayPath.Chmod(mode)
}

func (fs *BufferFS) Chown(name string, uid uint32, gid uint32, context *fuse.Context) (code fuse.Status) {
	// Do we need to do anything? Check the existing mode
	attr, status := fs.GetAttr(name, context)
	if status != fuse.OK {
		return status
	}
	if attr.Owner.Uid == uid && attr.Owner.Gid == gid {
		return fuse.OK
	}
	// The uid/gid will change, we need to overlay
	overlayPath := fs.overlay[name]
	if overlayPath == nil {
		if attr.IsDir() {
			overlayPath = NewOverlayDir(fs, name, 0, context)
		}
		if attr.IsRegular() {
			overlayPath = NewOverlayFile(fs, name, 0, 0, context)
		}
		if attr.IsSymlink() {
			target, st := fs.Readlink(name, context)
			if st != fuse.OK {
				return st
			}
			overlayPath = NewOverlaySymlink(fs, name, target, context)
		}
		fs.overlay[name] = overlayPath
	}
	return overlayPath.Chown(uid, gid)
}

func (fs *BufferFS) Truncate(path string, offset uint64, context *fuse.Context) (code fuse.Status) {
	overlayPath, status := fs.Open(path, fuse.W_OK, context)
	if status != fuse.OK {
		return status
	}
	return overlayPath.Truncate(offset)
}

func (fs *BufferFS) Readlink(name string, context *fuse.Context) (out string, code fuse.Status) {
	overlayPath := fs.overlay[name]
	if overlayPath != nil {
		return overlayPath.Target()
	}
	return fs.wrappedFS.Readlink(name, context)
}

func (fs *BufferFS) Unlink(name string, context *fuse.Context) (code fuse.Status) {
	// remove the entry in the parent dir
	dirname, basename := pathSplit(name)
	parent := fs.overlay[dirname]
	if parent == nil {
		parent = NewOverlayDir(fs, dirname, 0, context)
		fs.overlay[dirname] = parent
	}
	parent.RemoveEntry(basename)
	// unmap
	delete(fs.overlay, name)
	return fuse.OK
}

func (fs *BufferFS) Rmdir(name string, context *fuse.Context) (code fuse.Status) {
	// remove the entry in the parent dir
	dirname, basename := pathSplit(name)
	parent := fs.overlay[dirname]
	if parent == nil {
		parent = NewOverlayDir(fs, dirname, 0, context)
		fs.overlay[dirname] = parent
	}
	parent.RemoveEntry(basename)
	// unmap
	delete(fs.overlay, name)
	return fuse.OK
}

func (fs *BufferFS) Symlink(target string, name string, context *fuse.Context) (code fuse.Status) {
	// map
	child := NewOverlaySymlink(fs, name, target, context)
	fs.overlay[name] = child

	// create the entry in the parent dir
	dirname, basename := pathSplit(name)
	parent := fs.overlay[dirname]
	if parent == nil {
		parent = NewOverlayDir(fs, dirname, 0, context)
		fs.overlay[dirname] = parent
	}
	parent.AddEntry(fuse.S_IFLNK|0777, basename)
	return fuse.OK
}

func (fs *BufferFS) Mkdir(name string, mode uint32, context *fuse.Context) (code fuse.Status) {
	// map
	child := NewOverlayDir(fs, name, mode, context)
	fs.overlay[name] = child

	// create the entry in the parent dir
	dirname, basename := pathSplit(name)
	parent := fs.overlay[dirname]
	if parent == nil {
		parent = NewOverlayDir(fs, dirname, 0, context)
		fs.overlay[dirname] = parent
	}
	parent.AddEntry(fuse.S_IFDIR|mode, basename)
	return fuse.OK
}

func (fs *BufferFS) Create(name string, flags uint32, mode uint32, context *fuse.Context) (fuseFile nodefs.File, code fuse.Status) {
	// map
	child := NewOverlayFile(fs, name, flags, mode, context)
	fs.overlay[name] = child

	// create the entry in the parent dir
	dirname, basename := pathSplit(name)
	parent := fs.overlay[dirname]
	if parent == nil {
		parent = NewOverlayDir(fs, dirname, 0, context)
		fs.overlay[dirname] = parent
	}
	parent.AddEntry(fuse.S_IFREG|mode, basename)
	return child, fuse.OK
}

func (fs *BufferFS) GetOverlay(name string, context *fuse.Context) (res OverlayPath, code fuse.Status) {
	res = fs.overlay[name]
	if res == nil {
		attr, code := fs.GetAttr(name, context)
		if code != fuse.OK {
			return nil, code
		}
		if attr.IsDir() {
			res = NewOverlayDir(fs, name, attr.Mode, context)
		}
		if attr.IsRegular() {
			res = NewOverlayFile(fs, name, 0, attr.Mode, context)
		}
		if attr.IsSymlink() {
			target, st := fs.Readlink(name, context)
			if st != fuse.OK {
				return nil, st
			}
			res = NewOverlaySymlink(fs, name, target, context)
		}
	}
	return res, fuse.OK
}

func (fs *BufferFS) Rename(oldPath string, newPath string, context *fuse.Context) (code fuse.Status) {
	// TODO: Fuse checks existence of oldPath and the dir of the new path
	// for us. It does not check access.
	overlayPath, _ := fs.GetOverlay(oldPath, context)

	oldDir, oldBase := pathSplit(oldPath)
	oldParent, _ := fs.GetOverlay(oldDir, context)

	newDir, newBase := pathSplit(newPath)
	newParent, _ := fs.GetOverlay(newDir, context)

	// Map the new path
	fs.overlay[newPath] = overlayPath
	// Install the new entry in its parent
	attr := fuse.Attr{}
	overlayPath.GetAttr(&attr)
	newParent.AddEntry(attr.Mode, newBase)
	// If are moving a dir, we need to also remap the children before
	// unmapping the parent
	if attr.IsDir() {
		entries, status := fs.OpenDir(oldPath, context)
		if status != fuse.OK {
			return status
		}
		for _, e := range entries {
			toRename := path.Join(oldPath, e.Name)
			dest := path.Join(newPath, e.Name)
			status := fs.Rename(toRename, dest, context)
			if status != fuse.OK {
				return status
			}
		}
	}

	if oldParent != newParent || oldBase != newBase {
		oldParent.RemoveEntry(oldBase)
	}

	// Unmap the OverlayPath from its old path
	delete(fs.overlay, oldPath)
	return fuse.OK
}

func (fs *BufferFS) Access(name string, mode uint32, context *fuse.Context) (code fuse.Status) {
	attr, code := fs.GetAttr(name, context)
	if code != fuse.OK {
		return code
	}
	ownerbits := uint(6)
	groupbits := uint(3)
	otherbits := uint(0)
	b := otherbits
	if attr.Owner.Gid == context.Owner.Gid {
		b = groupbits
	}
	if attr.Owner.Uid == context.Owner.Uid {
		b = ownerbits
	}
	m := attr.Mode & 0777
	if m&(mode<<b) != 0 {
		return fuse.OK
	}
	return fuse.EACCES
}
