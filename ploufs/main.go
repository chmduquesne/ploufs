package main

import (
	"os"

	"github.com/chmduquesne/ploufs/fs"
)

func main() {
	fs.Mount(os.Args[1], os.Args[2])
}
