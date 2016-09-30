// Copyright 2016 Christophe-Marie Duquesne

package fs

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
)

func Mount(orig string, mountpoint string) {
	bindfs := NewBindFS(orig)
	bufferfs := NewBufferFS(bindfs)
	envVarExists := func(key string) bool { return os.Getenv(key) != "" }
	absolutePath := func(name string) string {
		res, _ := filepath.Abs(name)
		return res
	}
	envVarAsTokens := func(key string) []string {
		s := os.Getenv(key)
		if s == "" {
			return nil
		}
		return strings.Split(s, ",")
	}
	pathNodeFsOpts := &pathfs.PathNodeFsOptions{
		ClientInodes: envVarExists("ENABLE_LINKS"),
	}
	//pathFs := pathfs.NewPathNodeFs(bindfs, pathNodeFsOpts)
	pathFs := pathfs.NewPathNodeFs(bufferfs, pathNodeFsOpts)
	mountOpts := &fuse.MountOptions{
		Options:        envVarAsTokens("MOUNT_OPTIONS"),
		Name:           path.Base(os.Args[0]),
		FsName:         absolutePath(orig),
		Debug:          envVarExists("DEBUG"),
		SingleThreaded: envVarExists("SINGLE_THREADED"),
	}
	nodefsOpts := &nodefs.Options{
		NegativeTimeout: time.Second,
		AttrTimeout:     time.Second,
		EntryTimeout:    time.Second,
	}
	conn := nodefs.NewFileSystemConnector(pathFs.Root(), nodefsOpts)
	state, err := fuse.NewServer(conn.RawFS(), mountpoint, mountOpts)
	if err != nil {
		fmt.Printf("Mount fail: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Mounted!")
	state.Serve()
}
