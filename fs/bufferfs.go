// copyright 2016 Christophe-Marie Duquesne

package fs

import (
	"path"
	"sync"
	"time"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
)

type BufferFS struct {
	// We want a default implementation that fails for compile reasons
	pathfs.FileSystem
	// We also want a wrapped target, but we don't rely on its
	// implementation by default
	Wrapped   pathfs.FileSystem
	Overlayed map[string]OverlayPath
	lock      sync.Mutex
}

func pathSplit(name string) (dir string, base string) {
	dir, base = path.Split(name)
	if dir != "" {
		dir = dir[:len(dir)-1] // remove trailing '/' if present
	}
	return
}

func NewBufferFS(wrapped pathfs.FileSystem) pathfs.FileSystem {
	return &BufferFS{
		FileSystem: pathfs.NewDefaultFileSystem(),
		Wrapped:    wrapped,
		Overlayed:  make(map[string]OverlayPath),
	}
}

func (fs *BufferFS) Locked() func() {
	fs.lock.Lock()
	return func() { fs.lock.Unlock() }
}

func (fs *BufferFS) StatFs(name string) *fuse.StatfsOut {
	// We rely entirely on the underlying FS
	return fs.Wrapped.StatFs(name)
}

func (fs *BufferFS) OnMount(nodeFs *pathfs.PathNodeFs) {}

func (fs *BufferFS) OnUnmount() {}

func (fs *BufferFS) GetAttr(name string, context *fuse.Context) (a *fuse.Attr, code fuse.Status) {
	if name != "" {
		// If a file is not listed in its parent directory, it does not exist
		// (except for the root directory which does not list itself)
		dir, base := pathSplit(name)
		entries, status := fs.OpenDir(dir, context)
		if status != fuse.OK {
			// We could not open the parent
			return nil, status
		} else {
			found := false
			for _, e := range entries {
				if e.Name == base {
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
	// The file exists, but we may have overlayed it
	overlayPath := fs.Overlayed[name]
	if overlayPath != nil {
		a = &fuse.Attr{}
		code = overlayPath.GetAttr(a)
		return
	}
	// The file is not overlayed, we resort to the underlying file system
	a, code = fs.Wrapped.GetAttr(name, context)
	return
}

func (fs *BufferFS) OpenDir(name string, context *fuse.Context) (stream []fuse.DirEntry, status fuse.Status) {
	overlayPath := fs.Overlayed[name]
	if overlayPath != nil {
		return overlayPath.Entries(context)
	}
	return fs.Wrapped.OpenDir(name, context)
}

func (fs *BufferFS) OverlayFile(name string, mode uint32, context *fuse.Context) OverlayPath {
	overlayPath := fs.Overlayed[name]
	if overlayPath == nil {
		//log.Printf("Creating OverlayFile('%v')", name)
		attr := NewOverlayAttrFromScratch(fuse.S_IFREG|mode, context.Uid, context.Gid)
		source := NoSource
		a, code := fs.GetAttr(name, context)
		if code == fuse.OK {
			attr = NewOverlayAttrFromExisting(a)
			source = name
		}
		overlayPath = NewOverlayFile(attr, source)
		fs.Overlayed[name] = overlayPath
	}
	return overlayPath
}

func (fs *BufferFS) OverlayDir(name string, mode uint32, context *fuse.Context) OverlayPath {
	overlayPath := fs.Overlayed[name]
	if overlayPath == nil {
		//log.Printf("Creating OverlayDir('%v')", name)
		attr := NewOverlayAttrFromScratch(fuse.S_IFDIR|mode, context.Uid, context.Gid)
		entries := make([]fuse.DirEntry, 0)
		a, code := fs.GetAttr(name, context)
		if code == fuse.OK {
			attr = NewOverlayAttrFromExisting(a)
			entries, _ = fs.OpenDir(name, context)
		}
		overlayPath = NewOverlayDir(attr, entries)
		fs.Overlayed[name] = overlayPath
	}
	return overlayPath
}

func (fs *BufferFS) OverlaySymlink(name string, target string, context *fuse.Context) OverlayPath {
	overlayPath := fs.Overlayed[name]
	if overlayPath == nil {
		//log.Printf("Creating OverlaySymlink('%v')", name)
		attr := NewOverlayAttrFromScratch(fuse.S_IFLNK|0777, context.Uid, context.Gid)
		existingTarget, code := fs.Readlink(name, context)
		if code == fuse.OK {
			target = existingTarget
		}
		overlayPath = NewOverlaySymlink(attr, target)
		fs.Overlayed[name] = overlayPath
	}
	return overlayPath
}

func (fs *BufferFS) Open(name string, flags uint32, context *fuse.Context) (nodefs.File, fuse.Status) {
	// Assumes that fuse has checked the permissions
	overlayPath := fs.OverlayFile(name, 0, context)
	return NewOverlayFH(overlayPath, context, fs.Wrapped), fuse.OK
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
	overlayPath := fs.Overlayed[name]
	if overlayPath == nil {
		if attr.IsDir() {
			overlayPath = fs.OverlayDir(name, 0, context)
		}
		if attr.IsRegular() {
			overlayPath = fs.OverlayFile(name, 0, context)
		}
		// Permissions on symlinks don't make sense (I think) -> TESTME
		if attr.IsSymlink() {
			return fuse.OK
		}
	}
	return overlayPath.Chmod(mode)
}

func (fs *BufferFS) Chown(name string, uid uint32, gid uint32, context *fuse.Context) (code fuse.Status) {
	// Do we need to do anything? Check the existing mode
	attr, status := fs.GetAttr(name, context)
	if status != fuse.OK {
		return status
	}
	//log.Printf("uid: %v -> %v, gid: %v -> %v\n",
	//	attr.Owner.Uid, uid, attr.Owner.Gid, gid)
	if attr.Owner.Uid == uid && attr.Owner.Gid == gid {
		return fuse.OK
	}
	// The uid/gid will change, we need to OverlayedPaths
	overlayPath := fs.Overlayed[name]
	if overlayPath == nil {
		if attr.IsDir() {
			overlayPath = fs.OverlayDir(name, 0, context)
		}
		if attr.IsRegular() {
			overlayPath = fs.OverlayFile(name, 0, context)
		}
		if attr.IsSymlink() {
			overlayPath = fs.OverlaySymlink(name, "", context)
		}
	}
	return overlayPath.Chown(uid, gid)
}

func (fs *BufferFS) Truncate(path string, offset uint64, context *fuse.Context) (code fuse.Status) {
	overlayFH, status := fs.Open(path, fuse.W_OK, context)
	if status != fuse.OK {
		return status
	}
	return overlayFH.Truncate(offset)
}

func (fs *BufferFS) Readlink(name string, context *fuse.Context) (out string, code fuse.Status) {
	overlayPath := fs.Overlayed[name]
	if overlayPath != nil {
		return overlayPath.Target()
	}
	return fs.Wrapped.Readlink(name, context)
}

func (fs *BufferFS) Unlink(name string, context *fuse.Context) (code fuse.Status) {
	// remove the entry in the parent dir
	dir, base := pathSplit(name)
	parent := fs.OverlayDir(dir, 0, context)
	parent.RemoveEntry(base)
	// unmap
	delete(fs.Overlayed, name)
	return fuse.OK
}

func (fs *BufferFS) Rmdir(name string, context *fuse.Context) (code fuse.Status) {
	// remove the entry in the parent dir
	dir, base := pathSplit(name)
	parent := fs.OverlayDir(dir, 0, context)
	parent.RemoveEntry(base)
	// unmap
	delete(fs.Overlayed, name)
	return fuse.OK
}

func (fs *BufferFS) Symlink(target string, name string, context *fuse.Context) (code fuse.Status) {
	// map
	fs.OverlaySymlink(name, target, context)

	// create the entry in the parent dir
	dir, base := pathSplit(name)
	parent := fs.OverlayDir(dir, 0, context)
	parent.AddEntry(fuse.S_IFLNK|0777, base)
	return fuse.OK
}

func (fs *BufferFS) Mkdir(name string, mode uint32, context *fuse.Context) (code fuse.Status) {
	// map
	fs.OverlayDir(name, mode, context)

	// create the entry in the parent dir
	dir, base := pathSplit(name)
	parent := fs.OverlayDir(dir, 0, context)
	parent.AddEntry(fuse.S_IFDIR|mode, base)
	return fuse.OK
}

func (fs *BufferFS) Create(name string, flags uint32, mode uint32, context *fuse.Context) (fuseFile nodefs.File, code fuse.Status) {
	// map
	child := fs.OverlayFile(name, mode, context)

	// create the entry in the parent dir
	dir, base := pathSplit(name)
	parent := fs.OverlayDir(dir, 0, context)
	parent.AddEntry(fuse.S_IFREG|mode, base)
	return NewOverlayFH(child, context, fs.Wrapped), fuse.OK
}

func (fs *BufferFS) Rename(oldPath string, newPath string, context *fuse.Context) (code fuse.Status) {
	// TODO: Fuse checks existence of oldPath and the dir of the new path
	// for us. It does not check access.
	overlayPath := fs.Overlayed[oldPath]
	if overlayPath == nil {
		attr, _ := fs.GetAttr(oldPath, context)
		if attr.IsDir() {
			overlayPath = fs.OverlayDir(oldPath, 0, context)
		}
		if attr.IsRegular() {
			overlayPath = fs.OverlayFile(oldPath, 0, context)
		}
		if attr.IsSymlink() {
			overlayPath = fs.OverlaySymlink(oldPath, "", context)
		}
	}

	oldDir, oldBase := pathSplit(oldPath)
	oldParent := fs.OverlayDir(oldDir, 0, context)

	newDir, newBase := pathSplit(newPath)
	newParent := fs.OverlayDir(newDir, 0, context)

	// Map the new path
	fs.Overlayed[newPath] = overlayPath
	// Install the new entry in its parent
	attr := fuse.Attr{}
	overlayPath.GetAttr(&attr)
	newParent.RemoveEntry(newBase)
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
	delete(fs.Overlayed, oldPath)
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

func (fs *BufferFS) Utimens(name string, atime *time.Time, mtime *time.Time, context *fuse.Context) (code fuse.Status) {
	overlayPath := fs.Overlayed[name]
	if overlayPath == nil {
		attr, _ := fs.GetAttr(name, context)
		if attr.IsDir() {
			overlayPath = fs.OverlayDir(name, 0, context)
		}
		if attr.IsRegular() {
			overlayPath = fs.OverlayFile(name, 0, context)
		}
		if attr.IsSymlink() {
			overlayPath = fs.OverlaySymlink(name, "", context)
		}
	}
	return overlayPath.Utimens(atime, mtime)
}
