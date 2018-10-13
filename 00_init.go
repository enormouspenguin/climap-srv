package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	_STATIC_SUBDIR_NM = "static"
	_PIC_SUBDIR_NM    = "pic"
)

var (
	_EXE_PATH    string
	_ROOT_PATH   string
	_STATIC_PATH string
	_PIC_PATH    string
)

func init() {

	var err error

	_EXE_PATH, err = os.Executable()
	if err != nil {
		fmt.Println("UNABLE to get executable path!!!")
		panic(err)
	}

	_ROOT_PATH = filepath.Dir(_EXE_PATH)
	_STATIC_PATH = filepath.Join(_ROOT_PATH, _STATIC_SUBDIR_NM)
	_PIC_PATH = filepath.Join(_STATIC_PATH, _PIC_SUBDIR_NM)

	fi, err := os.Stat(_ROOT_PATH)
	if err != nil {
		fmt.Println("UNABLE to get stat of \"" + _ROOT_PATH + "\"!!! See more below:")
		panic(err)
	}

	err = os.MkdirAll(_STATIC_PATH, fi.Mode())
	if err != nil {
		fmt.Println("UNABLE to create static dir \"" + _STATIC_PATH + "\"!!! See more below:")
		panic(err)
	}

	init_pg()
	init_chi()
}
