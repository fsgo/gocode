// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/7/31

package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/fsgo/cmdutil"
	"github.com/fsgo/gomodule"
)

var input = flag.String("f", "modules.txt", "module list file path")
var outDir = flag.String("d", "./modules_download", "output dir")

func main() {
	flag.Parse()
	content, err := os.ReadFile(*input)
	if err != nil {
		log.Fatalln(err)
	}
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		download(line)
	}
}

func download(module string) {
	log.Println("downloading ", module)

	exportDir := filepath.Join(*outDir, module)
	st, err := os.Stat(exportDir)
	if err == nil && st.IsDir() {
		log.Println(color.YellowString("already exists: %s", exportDir))
		return
	}

	md := &gomodule.GoProxy{
		Module: module,
	}
	info, err := md.Latest(context.Background())
	if err != nil {
		log.Println(module, "Latest error:", err)
		return
	}
	log.Println(module, "Latest version=", info.Version)
	zrd, err := md.VersionZip(context.Background(), info.Version)
	if err != nil {
		log.Println(module, "VersionZip error:", err)
		return
	}
	tmpDir := exportDir + "_tmp"
	os.RemoveAll(tmpDir)
	err = os.MkdirAll(tmpDir, 0777)
	if err != nil {
		log.Println(module, "MkdirAll error:", err)
		return
	}
	zr := &cmdutil.Zip{
		StripComponents: uint(strings.Count(module, "/")) + 1,
	}
	err = zr.UnpackFromReader(zrd, tmpDir)
	if err != nil {
		log.Println(module, "UnpackFromReader error:", err)
		return
	}
	os.Rename(tmpDir, exportDir)
}
