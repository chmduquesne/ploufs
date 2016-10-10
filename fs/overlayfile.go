// copyright 2016 Christophe-Marie Duquesne

package fs

import (
	"fmt"
	"log"
	"sync"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

const (
	NoSource = "/"
)

type OverlayFile struct {
	File
	Dir
	Symlink
	Attr
	fs      *BufferFS
	source  string
	slices  []*FileSlice
	context *fuse.Context
	lock    sync.Mutex
}

func NewOverlayFile(fs *BufferFS, source string, flags uint32, mode uint32, context *fuse.Context) OverlayPath {
	log.Printf("Creating overlay file for '%s'\n", source)
	_, status := fs.GetAttr(source, context)
	if status != fuse.OK {
		source = NoSource
		log.Printf("Underlying file system reports no source")
	}
	b := &OverlayFile{
		File:    nodefs.NewDefaultFile(),
		Dir:     NewDefaultDir(),
		Symlink: NewDefaultSymlink(),
		Attr:    NewAttr(fs, source, fuse.S_IFREG|mode, context),
		fs:      fs,
		source:  source,
		context: context,
	}
	return b
}

func (f *OverlayFile) String() string {
	return fmt.Sprintf("OverlayFile{}")
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
		f.SetSize(offset)
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
		file, status := f.fs.wrappedFS.Open(f.source, fuse.R_OK, f.context)
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
		f.SetSize(eow)
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
