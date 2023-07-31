// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/7/31

package main

import (
	"bytes"
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fsgo/gomodule"
	"golang.org/x/mod/modfile"
)

var srcDir = flag.String("src", "modules_download", "scan dir")
var ignoreErr = flag.Bool("ig", false, "ignore error")
var outDir = flag.String("d", "scan_result", "result dir")

var command = flag.String("cmd", "go-doc-json ./...", "command to execute")

func main() {
	flag.Parse()
	if *srcDir == "" {
		log.Fatalln("empty src dir")
	}
	_ = os.MkdirAll(*outDir, 0777)
	err := gomodule.ScanGoModFile(*srcDir, func(dir string, mod modfile.File) error {
		err := doModule(dir, mod)
		if err != nil && *ignoreErr {
			log.Println("Dir:", dir, "Has error:", err)
			return nil
		}
		return err
	})
	log.Println("scan result:", err)
}

func doModule(dir string, mod modfile.File) error {
	if *command == "" {
		log.Fatalln("cmd is empty")
	}
	arr := strings.Fields(*command)
	cmd := exec.Command(arr[0], arr[1:]...)
	log.Println("Dir:", dir, "Module:", mod.Module.Mod.Path, "Exec:", cmd.String())
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	bf := &bytes.Buffer{}
	cmd.Stdout = bf
	err := cmd.Run()
	if err != nil {
		return err
	}
	outFilePath := filepath.Join(*outDir, strings.ReplaceAll(mod.Module.Mod.Path, "/", "_")+".jsonl")
	return os.WriteFile(outFilePath, bf.Bytes(), 0644)
}
