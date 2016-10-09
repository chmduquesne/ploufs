// copyright 2016 Christophe-Marie Duquesne

package fs

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
)

const (
	NoSource = "/"
)

type OverlayFile struct {
	nodefs.File
	wrappedFS pathfs.FileSystem
	source    string
	attr      *fuse.Attr
	slices    []*FileSlice
	entries   []fuse.DirEntry
	deleted   bool
	context   *fuse.Context
	lock      sync.Mutex
}

func NewOverlayFile(wrappedFS pathfs.FileSystem, source string, context *fuse.Context) *OverlayFile {
	log.Printf("Creating overlay file for %s\n", source)
	attr, status := wrappedFS.GetAttr(source, context)
	if status != fuse.OK {
		source = NoSource
		log.Printf("Underlying file system reports no source", source)
	}
	b := &OverlayFile{
		File:      nodefs.NewDefaultFile(),
		wrappedFS: wrappedFS,
		source:    source,
		attr:      attr,
		context:   context,
	}
	return b
}

func (f *OverlayFile) String() string {
	return fmt.Sprintf("OverlayFile{\nsource: %s\n	attr: %v\n	n_slices: %v\n	deleted: %v\n}", f.source, f.attr, len(f.slices), f.deleted)
}

func (f *OverlayFile) GetAttr(out *fuse.Attr) (code fuse.Status) {
	if f.deleted {
		return fuse.ENOENT
	}
	out.Ino = f.attr.Ino
	out.Size = f.attr.Size
	out.Blocks = f.attr.Blocks
	out.Atime = f.attr.Atime
	out.Mtime = f.attr.Mtime
	out.Ctime = f.attr.Ctime
	out.Atimensec = f.attr.Atimensec
	out.Mtimensec = f.attr.Mtimensec
	out.Ctimensec = f.attr.Ctimensec
	out.Mode = f.attr.Mode
	out.Nlink = f.attr.Nlink
	out.Owner = f.attr.Owner
	out.Rdev = f.attr.Rdev
	out.Blksize = f.attr.Blksize
	out.Padding = f.attr.Padding
	return fuse.OK
}

func (f *OverlayFile) Size() uint64 {
	return f.attr.Size
}

func (f *OverlayFile) Truncate(offset uint64) fuse.Status {
	f.lock.Lock()
	defer f.lock.Unlock()

	off := int64(offset)
	slices := make([]*FileSlice, 0)
	// Remove all the slices after the truncation
	for _, s := range f.slices {
		// the slice is entirely before the truncation
		if s.Beg() <= off && s.End() <= off {
			slices = append(slices, s)
		}
		// this slice is truncated
		if s.Beg() <= off && s.End() > off {
			// cut the slice
			slices = append(slices, s.Truncated(off))
		}
	}
	f.slices = slices

	if offset <= f.Size() {
		// The cut shortens the file
		// We don't need to create an extra slice because we will always
		// be able to read whatever already exists
		f.attr.Size = offset
	} else {
		// man 2 truncate says we need to extend the file with 0 which is
		// equivalent to a call to write from the end of the file to the
		// specified new end.
		buf := make([]byte, offset-f.Size())
		f.Write(buf, int64(f.Size()))
	}
	return fuse.OK
}

func (f *OverlayFile) SetInode(*nodefs.Inode) {}

//func (f *OverlayFile) InnerFile() nodefs.File {
//	return f.File
//}

func (f *OverlayFile) Read(buf []byte, off int64) (fuse.ReadResult, fuse.Status) {
	f.lock.Lock()
	defer f.lock.Unlock()

	n := len(buf)
	res := &FileSlice{
		offset: off,
		data:   buf[:0],
	}

	// man 2 read: If the file offset is at or past the end of file, no
	// bytes are read, and read() returns zero (== fuse.OK for us)
	if uint64(off) >= f.Size() {
		return res, fuse.OK
	}

	// First, read what we want from the wrapped file
	if f.source != NoSource {
		file, status := f.wrappedFS.Open(f.source, fuse.R_OK, f.context)
		if status != fuse.OK {
			log.Fatalf("Could not open the underlying file in read mode\n")
		}
		r, status := file.Read(buf, off)
		if status != fuse.OK {
			log.Fatalf("Could not read the underlying file\n")
		}
		b, _ := r.Bytes(buf)
		res.data = b
		file.Release()
	}

	// Merge all overlapping existing data into the result
	for _, s := range f.slices {
		if res.Overlaps(s) {
			res = res.BringIn(s)
		}
	}

	res = res.Shortened(n)

	return res, fuse.OK
}

func (f *OverlayFile) Write(data []byte, off int64) (uint32, fuse.Status) {
	f.lock.Lock()
	defer f.lock.Unlock()

	toInsert := &FileSlice{
		data:   data,
		offset: off,
	}

	// Merge all overlapping slices together
	for _, s := range f.slices {
		if s.Overlaps(toInsert) {
			toInsert = s.BringIn(toInsert)
		}
	}

	// Keep the slice sorted by offset and non overlapping
	slices := make([]*FileSlice, 0)
	isInserted := false
	for _, s := range f.slices {
		if !s.Overlaps(toInsert) {
			// insert every non-overlapping slice
			if s.Beg() > toInsert.Beg() && !isInserted {
				// insert our slice before any other starting after
				slices = append(slices, toInsert)
				isInserted = true
			}
			slices = append(slices, s)
		}
	}
	// If we did not insert the slice previously, do it now
	if !isInserted {
		slices = append(slices, toInsert)
	}
	f.slices = slices

	// Update the file size if needed
	eow := uint64(int(off) + len(data))
	if eow > f.Size() {
		f.attr.Size = eow
	}

	return uint32(len(data)), fuse.OK
}

func (f *OverlayFile) Release() {
	// Do we want to do something?
}

func (f *OverlayFile) Flush() fuse.Status {
	// Report success, but actually we will wait for sync to do anyting
	return fuse.OK
}

func (f *OverlayFile) Fsync(flags int) (code fuse.Status) {
	// TODO: implement this
	return fuse.OK
}

func (f *OverlayFile) Utimens(a *time.Time, m *time.Time) fuse.Status {
	// TODO: implement this
	return fuse.OK
}

func (f *OverlayFile) Allocate(off uint64, sz uint64, mode uint32) fuse.Status {
	// This filesystem does not offer any guarantee that the changes will
	// be written.
	return fuse.ENOSYS
}

func (f *OverlayFile) Chmod(mode uint32) fuse.Status {
	f.attr.Mode = (f.attr.Mode & 0xfe00) | mode
	return fuse.OK
}

func (f *OverlayFile) Chown(uid uint32, gid uint32) fuse.Status {
	f.attr.Uid = uid
	f.attr.Gid = gid
	return fuse.OK
}
