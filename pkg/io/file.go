// Package io contains various Input Output functions used within the watcher application
package io

import (
	"fmt"
	"io"
	"os"

	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

// CopyFile copies a file from the given src path to the given destination path and returns possible occurred errors
func CopyFile(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	// nolint: gosec
	source, err := os.Open(src)
	if err != nil {
		return err
	}

	defer raven.CheckClosure(source)

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}

	defer raven.CheckClosure(destination)

	_, err = io.Copy(destination, source)

	return err
}
