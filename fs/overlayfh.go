package fs

import "github.com/hanwen/go-fuse/fuse"

type OverlayFH struct {
	OverlayPath
	context *fuse.Context
}

func NewOverlayFH(o OverlayPath, context *fuse.Context) *OverlayFH {
	return &OverlayFH{
		OverlayPath: o,
		context:     context,
	}
}

func (h *OverlayFH) Read(dest []byte, off int64) (fuse.ReadResult, fuse.Status) {
	return h.OverlayPath.Read(dest, off, h.context)
}

func (h *OverlayFH) Write(data []byte, off int64) (uint32, fuse.Status) {
	return h.OverlayPath.Write(data, off, h.context)
}
