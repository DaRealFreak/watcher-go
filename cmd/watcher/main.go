package main

import (
	"github.com/kubernetes/klog"
	"watcher-go/cmd/watcher/cmd"
)

func init() {
	klog.InitFlags(nil)
}

func main() {
	cmd.Execute()
}
