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
	return fmt.Sprintf("OverlayFile{\nsource: %s	attr: %v\n	n_slices: %v\n	deleted: %v\n}", f.source, f.attr, len(f.slices), f.deleted)
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
			truncated := &FileSlice{
				offset: s.offset,
				data:   s.data[:int64(len(s.data))+s.offset-off],
			}
			slices = append(slices, truncated)
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

	// man 2 read: If the file offset is at or past the end of file, no
	// bytes are read, and read() returns zero
	if uint64(off) >= f.Size() {
		return fuse.ReadResultData(make([]byte, 0)), fuse.OK
	}

	b := make([]byte, len(buf))
	// First, read what we want from the wrapped file
	if f.source != NoSource {
		file, status := f.wrappedFS.Open(f.source, fuse.R_OK, f.context)
		if status != fuse.OK {
			log.Println("Could not open the underlying file in read mode")
		}
		file.Read(b, off)
		file.Release()
	}

	// Bring in the result into a Fileslice
	slice := &FileSlice{
		data:   b,
		offset: off,
	}

	// Merge all overlapping existing data into the result
	for _, s := range f.slices {
		if s.Overlaps(slice) {
			slice = slice.Merge(s)
		}
	}

	log.Printf("File: %v\n", f)
	log.Printf("Merged slices: %v\n", slice)

	// Copy whatever has been brought in
	n := copy(buf, slice.data)
	res := fuse.ReadResultData(buf[:n])

	return res, fuse.OK
}

func (f *OverlayFile) Write(data []byte, off int64) (uint32, fuse.Status) {
	f.lock.Lock()
	defer f.lock.Unlock()

	toInsert := &FileSlice{
		data:   data,
		offset: off,
	}
	// Create the slice that merges all mergeable slices
	for _, s := range f.slices {
		if s.Overlaps(toInsert) {
			toInsert = s.Merge(toInsert)
		}
	}

	// Keep the slice sorted by offset and non overlapping
	slices := make([]*FileSlice, 0)
	inserted := false
	for _, s := range f.slices {
		if !s.Overlaps(toInsert) {
			if s.offset > toInsert.offset && !inserted {
				slices = append(slices, toInsert)
				inserted = true
			}
			slices = append(slices, s)
		}
	}
	f.slices = slices

	// Update the file size if needed
	endOfWrite := uint64(off) + uint64(len(data))
	if endOfWrite > f.Size() {
		f.attr.Size = endOfWrite
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
