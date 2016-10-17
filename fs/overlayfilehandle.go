package fs

import "github.com/hanwen/go-fuse/fuse"

type OverlayFileHandle struct {
	OverlayPath
	context *fuse.Context
}

func NewOverlayFileHandle(o OverlayPath, context *fuse.Context) *OverlayFileHandle {
	return &OverlayFileHandle{
		OverlayPath: o,
		context:     context,
	}
}

func (h *OverlayFileHandle) Read(dest []byte, off int64) (fuse.ReadResult, fuse.Status) {
	return h.OverlayPath.Read(dest, off, h.context)
}

func (h *OverlayFileHandle) Write(data []byte, off int64) (uint32, fuse.Status) {
	return h.OverlayPath.Write(data, off, h.context)
}
