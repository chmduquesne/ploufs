// copyright 2016 Christophe-Marie Duquesne

package fs

import (
	"fmt"
	"sync"
	"time"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type BufferFile struct {
	nodefs.File
	attr   *fuse.Attr
	slices []*FileSlice
	orig   string
	lock   sync.Mutex
}

// Bufferfile buffers all modifications made to the filesystem in memory
// in order to apply them later, in one single operation.
// It relies on another file implementation to get the existing data.
func NewBufferFile(wrapped nodefs.File) *BufferFile {
	b := &BufferFile{
		File:   wrapped,
		slices: nil,
	}
	return b
}

func (f *BufferFile) Size() uint64 {
	return f.attr.Size
}

func (f *BufferFile) Truncate(offset uint64) fuse.Status {
	f.lock.Lock()
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
		// man 2 truncate says we need to extend the file with 0
		buf := make([]byte, offset-f.Size())
		f.Write(buf, int64(f.Size()))
	}
	f.lock.Unlock()
	return fuse.OK
}

func (f *BufferFile) SetInode(*nodefs.Inode) {}

func (f *BufferFile) String() string {
	return fmt.Sprintf("BufferFile(%s)", f.File.String())
}

func (f *BufferFile) InnerFile() nodefs.File {
	return f.File
}

func (f *BufferFile) Read(buf []byte, off int64) (fuse.ReadResult, fuse.Status) {
	// It is assumed that the file exists
	f.lock.Lock()

	// First, read what we want from the wrapped file
	b := make([]byte, len(buf))
	f.File.Read(b, off)

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

	// Copy whatever has been brought in
	n := copy(buf, slice.data)
	res := fuse.ReadResultData(buf[:n])

	f.lock.Unlock()
	return res, fuse.OK
}

func (f *BufferFile) Write(data []byte, off int64) (uint32, fuse.Status) {
	f.lock.Lock()
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
	f.lock.Unlock()
	return uint32(len(data)), fuse.OK
}

func (f *BufferFile) Release() {
	// Do we want to do something?
}

func (f *BufferFile) Flush() fuse.Status {
	// Report success, but actually we will wait for sync to do anyting
	return fuse.OK
}

func (f *BufferFile) Fsync(flags int) (code fuse.Status) {
	// TODO: implement this
	return fuse.OK
}

func (f *BufferFile) Utimens(a *time.Time, m *time.Time) fuse.Status {
	// TODO: implement this
	return fuse.OK
}

func (f *BufferFile) Allocate(off uint64, sz uint64, mode uint32) fuse.Status {
	// This filesystem does not offer any guarantee that the changes will
	// be written.
	return fuse.ENOSYS
}
