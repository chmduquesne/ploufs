// copyright 2016 Christophe-Marie Duquesne

package fs

import (
	"fmt"
	"log"
	"sync"

	"github.com/hanwen/go-fuse/fuse"
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
		File:    NewDefaultFile(),
		Dir:     NewDefaultDir(),
		Symlink: NewDefaultSymlink(),
		Attr:    NewAttr(fs, source, fuse.S_IFREG|mode, context),
		fs:      fs,
		source:  source,
		context: context,
	}
	return b
}

func (f *OverlayFile) Locked() (unlock func()) {
	f.lock.Lock()
	return func() { f.lock.Unlock() }
}

func (f *OverlayFile) String() string {
	return fmt.Sprintf("OverlayFile{}")
}

func (f *OverlayFile) Truncate(offset uint64) fuse.Status {
	defer f.Locked()()

	off := int64(offset)
	slices := make([]*FileSlice, 0, len(f.slices)+1)
	// Remove all the slices after the truncation
	for _, s := range f.slices {
		// the slice is entirely before the truncation
		if s.End() <= off {
			slices = append(slices, s)
		}
		// this slice is truncated but not empty (no point in having an
		// empty slice)
		if s.Beg() < off && s.End() > off {
			// cut the slice
			slices = append(slices, s.Truncated(off))
		}
	}

	if offset > f.Size() {
		// The cut strictly extends the slice. man 2 truncate says we
		// need to extend the file with 0. We add a slice from the end of
		// the file.
		s := &FileSlice{
			offset: int64(f.Size()),
			data:   make([]byte, offset-f.Size()),
		}
		slices = append(slices, s)
	}

	f.slices = slices
	f.SetSize(offset)
	return fuse.OK
}

func (f *OverlayFile) Read(buf []byte, off int64, ctx *fuse.Context) (fuse.ReadResult, fuse.Status) {
	defer f.Locked()()

	res := &FileSlice{
		offset: off,
		data:   buf,
	}

	log.Printf("reading %v bytes from offset %v", len(buf), off)

	// man 2 read: If the file offset is at or past the end of file, no
	// bytes are read, and read() returns zero (== fuse.OK for us)
	if uint64(off) >= f.Size() {
		res := &FileSlice{
			offset: off,
			data:   buf[:0],
		}
		return res, fuse.OK
	}

	// First, read what we want from the wrapped file
	if f.source != NoSource {
		file, status := f.fs.Wrapped.Open(f.source, fuse.R_OK, ctx)
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
			res.Write(s)
		}
	}

	return res.Truncated(int64(f.Size())), fuse.OK
}

func (f *OverlayFile) Write(data []byte, off int64, ctx *fuse.Context) (uint32, fuse.Status) {
	log.Printf("writing %v bytes at offset %v", len(data), off)
	defer f.Locked()()

	d := make([]byte, len(data))
	copy(d, data)
	toInsert := &FileSlice{
		data:   d,
		offset: off,
	}

	// Possible optimization: Because most writes are append-only (write
	// to the end of the file), we should look for overlapping slices from
	// the end of the file in order to interrupt the lookup faster

	// Merge all overlapping slices together
	for _, s := range f.slices {
		if toInsert.Overlaps(s) {
			toInsert = s.MergedIn(toInsert)
		}
	}

	// Keep the slice sorted by offset and non overlapping
	slices := make([]*FileSlice, 0, len(f.slices)+1)
	isInserted := false
	for _, s := range f.slices {
		if !toInsert.Overlaps(s) {
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
