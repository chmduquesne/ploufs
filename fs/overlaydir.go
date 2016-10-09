// copyright 2016 Christophe-Marie Duquesne

package fs

import (
	"fmt"
	"log"
	"sync"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type OverlayDir struct {
	nodefs.File
	attr    *OverlayAttr
	entries []fuse.DirEntry
	lock    sync.Mutex
}

func NewOverlayDir(fs *BufferFS, path string, mode uint32, context *fuse.Context) *OverlayDir {
	log.Printf("Creating overlay dir for %s\n", path)
	entries, status := fs.OpenDir(path, context)
	if status != fuse.OK {
		entries = make([]fuse.DirEntry, 0)
	}
	d := &OverlayDir{
		File:    nodefs.NewDefaultFile(),
		attr:    NewOverlayAttr(fs, path, fuse.S_IFDIR|mode, context),
		entries: entries,
	}
	return d
}

func (d *OverlayDir) AddEntry(mode uint32, name string) (code fuse.Status) {
	for _, e := range d.entries {
		if e.Name == name {
			return
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

func (d *OverlayDir) GetAttr(out *fuse.Attr) (code fuse.Status) {
	return d.attr.GetAttr(out)
}

func (d *OverlayDir) Utimens(a *time.Time, m *time.Time) fuse.Status {
	return d.attr.Utimens(a, m)
}

func (d *OverlayDir) Chmod(mode uint32) fuse.Status {
	return d.attr.Chmod(mode)
}

func (d *OverlayDir) Chown(uid uint32, gid uint32) fuse.Status {
	return d.attr.Chown(uid, gid)
}

func (d *OverlayDir) Deleted() bool {
	return d.attr.Deleted()
}

func (d *OverlayDir) MarkDeleted() {
	d.attr.MarkDeleted()
}

func (d *OverlayDir) Target() (target string, code fuse.Status) {
	return "", fuse.ToStatus(syscall.ENOLINK)
}

func (d *OverlayDir) SetTarget(target string) (code fuse.Status) {
	return fuse.ToStatus(syscall.ENOLINK)
}
