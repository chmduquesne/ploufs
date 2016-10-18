// copyright 2016 Christophe-Marie Duquesne

package fs

import (
	"fmt"
	"sync"
	"syscall"

	"github.com/hanwen/go-fuse/fuse"
)

type OverlayDir struct {
	File
	Symlink
	OverlayAttr
	entries []fuse.DirEntry
	lock    sync.Mutex
}

func NewOverlayDir(attr OverlayAttr, entries []fuse.DirEntry) OverlayPath {
	return &OverlayDir{
		File:        NewDefaultFile(),
		Symlink:     NewDefaultSymlink(),
		OverlayAttr: attr,
		entries:     entries,
	}
}

func (d *OverlayDir) AddEntry(mode uint32, name string) (code fuse.Status) {
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
