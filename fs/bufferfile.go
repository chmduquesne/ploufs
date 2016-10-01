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
	slices []*FileSlice
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

func (f *BufferFile) SetInode(*nodefs.Inode) {}

func (f *BufferFile) String() string {
	return fmt.Sprintf("BufferFile(%s)", f.File.String())
}

func (f *BufferFile) InnerFile() nodefs.File {
	return f.File
}

func (f *BufferFile) Read(buf []byte, off int64) (fuse.ReadResult, fuse.Status) {
	f.lock.Lock()
	b := make([]byte, len(buf))
	f.File.Read(b, off)

	// Bring in the result of the Read() into a Fileslice
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

func (f *BufferFile) Release() {}

func (f *BufferFile) Flush() fuse.Status {
	return fuse.OK
}

func (f *BufferFile) Fsync(flags int) (code fuse.Status) {
	// TODO: actually write changes to disk
	return fuse.OK
}

func (f *BufferFile) Allocate(off uint64, sz uint64, mode uint32) fuse.Status {
	return fuse.OK
}

func (f *BufferFile) Utimens(a *time.Time, m *time.Time) fuse.Status {
	return fuse.OK
}
