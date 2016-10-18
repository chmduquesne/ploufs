package fs

import (
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/pathfs"
)

type OverlayFH struct {
	OverlayPath
	context *fuse.Context
	fs      pathfs.FileSystem
}

func NewOverlayFH(o OverlayPath, context *fuse.Context, fs pathfs.FileSystem) *OverlayFH {
	return &OverlayFH{
		OverlayPath: o,
		context:     context,
		fs:          fs,
	}
}

func (h *OverlayFH) Read(dest []byte, off int64) (fuse.ReadResult, fuse.Status) {
	return h.OverlayPath.Read(dest, off, h.context, h.fs)
}

func (h *OverlayFH) Write(data []byte, off int64) (uint32, fuse.Status) {
	return h.OverlayPath.Write(data, off, h.context, h.fs)
}
