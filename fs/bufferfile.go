// copyright 2016 Christophe-Marie Duquesne

package fs

import (
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type BufferFile struct {
	nodefs.File
	slices []*FileSlice
}

func NewBufferFile(wrapped nodefs.File) *BufferFile {
	b := &BufferFile{
		File:   wrapped,
		slices: nil,
	}
	return b
}

//func (f *BufferFile) String() string {
//	s := make([]string, 0)
//	for _, slice := range f.slices {
//		s = append(s, slice.String())
//	}
//	return fmt.Sprintf("BufferFile{%v}", strings.Join(s, ", "))
//}

func (f *BufferFile) Write(data []byte, off int64) (uint32, fuse.Status) {
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

	return uint32(len(data)), fuse.OK
}

func (f *BufferFile) Read(buf []byte, off int64) (fuse.ReadResult, fuse.Status) {
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

	return res, fuse.OK
}
