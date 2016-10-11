// copyright 2016 Christophe-Marie Duquesne

package fs

import (
	"fmt"
	"log"
	"sync"
	"syscall"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type OverlayDir struct {
	File
	Symlink
	Attr
	entries []fuse.DirEntry
	lock    sync.Mutex
	path    string
}

func NewOverlayDir(fs *BufferFS, path string, mode uint32, context *fuse.Context) OverlayPath {
	log.Printf("Creating overlay dir for '%s'\n", path)
	entries, status := fs.OpenDir(path, context)
	if status != fuse.OK {
		entries = make([]fuse.DirEntry, 0)
	}
	d := &OverlayDir{
		File:    nodefs.NewDefaultFile(),
		Symlink: NewDefaultSymlink(),
		Attr:    NewAttr(fs, path, fuse.S_IFDIR|mode, context),
		entries: entries,
		path:    path,
	}
	return d
}

func (d *OverlayDir) AddEntry(mode uint32, name string) (code fuse.Status) {
	log.Printf("%s.AttrEntry(%s)", d.path, name)
	for _, e := range d.entries {
		if e.Name == name {
			return fuse.ToStatus(syscall.EEXIST)
		}
	}
	e := fuse.DirEntry{
		Mode: mode,
		Name: name,
	}
	d.entries = append(d.entries, e)
	return fuse.OK
}

func (d *OverlayDir) RemoveEntry(name string) (code fuse.Status) {
	log.Printf("%s.RemoveEntry(%s)", d.path, name)
	entries := make([]fuse.DirEntry, 0, len(d.entries))
	for _, e := range d.entries {
		if e.Name != name {
			entries = append(entries, e)
		}
	}
	d.entries = entries
	return fuse.OK
}

func (d *OverlayDir) Entries(context *fuse.Context) (stream []fuse.DirEntry, code fuse.Status) {
	return d.entries, fuse.OK
}

func (d *OverlayDir) String() string {
	return fmt.Sprintf("OverlayDir{%v}", d.entries)
}
