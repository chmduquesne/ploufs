package fs

import (
	"io/ioutil"
	"os"
	"syscall"
	"testing"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
)

// We wrap testing.T to get an extra method Mkdir
type T struct {
	*testing.T
}

func NewT(t *testing.T) *T {
	return &T{T: t}
}

func (t *T) Mkdir(dirname string, mode os.FileMode) {
	if err := os.Mkdir(dirname, mode); err != nil {
		t.Fatalf("Mkdir(%q,%v): %v", dirname, mode, err)
	}
}

type FSImplem interface {
	Root() string
	Setup(dirname string)
	Clean()
	String() string
}

type TestFunc func(FSImplem, *T)

func TestAllImplem(wrapped *testing.T, test TestFunc) {
	t := NewT(wrapped)
	implementations := [2]FSImplem{NewNativeFSImplem(), NewBufferFSImplem(t)}
	for _, impl := range implementations {
		t.Logf("-- %v --\n", impl)

		// Make sure system setting does not affect test.
		syscall.Umask(0)

		tmpDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		impl.Setup(tmpDir)
		test(impl, t)
		impl.Clean()
		os.RemoveAll(tmpDir)
	}

}

type NativeFSImplem struct {
	root string
}

func NewNativeFSImplem() *NativeFSImplem {
	return &NativeFSImplem{}
}

func (implem *NativeFSImplem) Root() string {
	return implem.root
}

func (implem *NativeFSImplem) String() string {
	return "NativeFSImplem"
}

func (implem *NativeFSImplem) Setup(dirname string) {
	implem.root = dirname
}

func (implem *NativeFSImplem) Clean() {}

type BufferFSImplem struct {
	t         *T
	root      string
	state     *fuse.Server
	connector *nodefs.FileSystemConnector
}

func NewBufferFSImplem(t *T) FSImplem {
	return &BufferFSImplem{t: t}
}

func (implem *BufferFSImplem) Root() string {
	return implem.root
}

func (implem *BufferFSImplem) String() string {
	return "BufferFSImplem"
}

func (implem *BufferFSImplem) Setup(dirname string) {
	var err error

	ori := dirname + "/ori"
	os.Mkdir(ori, 0700)
	mnt := dirname + "/mnt"
	os.Mkdir(mnt, 0700)
	implem.root = mnt

	bfs := NewBufferFS(pathfs.NewLoopbackFileSystem(ori))
	pnfs := pathfs.NewPathNodeFs(bfs, &pathfs.PathNodeFsOptions{ClientInodes: true})
	implem.connector = nodefs.NewFileSystemConnector(pnfs.Root(),
		&nodefs.Options{})
	implem.state, err = fuse.NewServer(
		fuse.NewRawFileSystem(implem.connector.RawFS()), mnt, &fuse.MountOptions{
			SingleThreaded: true,
			//Debug:          VerboseTest(),
		})

	if err != nil {
		implem.t.Fatal("NewServer", err)
	}

	go implem.state.Serve()
	if err := implem.state.WaitMount(); err != nil {
		implem.t.Fatal("WaitMount", err)
	}
}

func (implem *BufferFSImplem) Clean() {
	err := implem.state.Unmount()
	if err != nil {
		implem.t.Fatalf("Unmount failed: %v\n", err)
	}
}
