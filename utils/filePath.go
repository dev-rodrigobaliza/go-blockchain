package utils

import (
	"os"
	"path/filepath"
)

const (
	systemPath = "tmp"
)

func CheckSystemPath() string {
	curPath, err := os.Getwd()
	Handle(err)

	systemPath := filepath.Join(curPath, systemPath)

	_ = os.Mkdir(systemPath, os.ModePerm)

	return systemPath
}
