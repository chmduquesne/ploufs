// Copyright 2016 Christophe-Marie Duquesne

package fs

import (
	"fmt"
	"os"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
)

func Mount(orig string, mountpoint string) {
	fs := NewFS(orig)
	pathFsOpts := &pathfs.PathNodeFsOptions{ClientInodes: false}
	pathFs := pathfs.NewPathNodeFs(fs, pathFsOpts)
	conn := nodefs.NewFileSystemConnector(pathFs.Root(), nil)
	state, err := fuse.NewServer(conn.RawFS(), mountpoint, nil)
	if err != nil {
		fmt.Printf("Mount fail: %v\n", err)
		os.Exit(1)
	}
	state.Serve()
}
