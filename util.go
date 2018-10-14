package main

import (
	"net/http"
	"os"
)

func newDirFS(root string) http.FileSystem {
	return noDirListingFS{http.Dir(root)}
}

type noDirListingFS struct {
	fs http.FileSystem
}

func (fs noDirListingFS) Open(name string) (http.File, error) {
	// fmt.Println(name)
	f, err := fs.fs.Open(name)
	if err != nil {
		return nil, err
	}
	return bluffReaddirFile{f}, nil
}

//pretend to support http.File.Readdir() to satisfy the http.File interface but always ignore the calls and returns nothing.
//this is for preventing directory listing from the outside.
type bluffReaddirFile struct {
	http.File
}

func (f bluffReaddirFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}
