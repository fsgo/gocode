// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/7/31

package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/fsgo/gomodule"
	"golang.org/x/mod/modfile"
)

var srcDir = flag.String("src", "modules_download", "scan dir")
var ignoreErr = flag.Bool("ig", true, "ignore error")
var outDir = flag.String("d", "scan_result", "result dir")
var command = flag.String("cmd", "go-doc-json -test=false ./...", "command to execute")
var conc = flag.Int("c", 2, "Number of multiple task to make at a time")

func main() {
	flag.Parse()
	if *srcDir == "" {
		log.Fatalln("empty src dir")
	}

	_ = os.MkdirAll(*outDir, 0777)

	var total int
	_ = gomodule.ScanGoModFile(*srcDir, func(dir string, mod modfile.File) error {
		total++
		return nil
	})

	var id atomic.Int64
	err := gomodule.ScanGoModFileParallel(*srcDir, *conc, func(dir string, mod modfile.File) error {
		idx := fmt.Sprintf("%d/%d", id.Add(1), total)
		err := doModule(idx, dir, mod)
		if err != nil && *ignoreErr {
			return nil
		}
		return err
	})
	log.Println("scan result:", err, ",failed:", failed.Load())
}

var failed atomic.Int64

func doModule(index string, dir string, mod modfile.File) (err error) {
	if *command == "" {
		log.Fatalln("cmd is empty")
	}
	start := time.Now()
	defer func() {
		cost := time.Since(start)
		if err == nil {
			log.Println(color.GreenString("[%s] Success", index), "Dir:", dir, "Cost:", cost.String())
		} else {
			failed.Add(1)
			log.Println(color.RedString("[%s] Failed", index), "Dir:", dir, "Cost:", cost.String(), "Err:", err)
		}
	}()
	arr := strings.Fields(*command)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, arr[0], arr[1:]...)
	idStr := color.GreenString("[%s]", index)
	log.Println(idStr, "Dir:", dir, "Module:", mod.Module.Mod.Path, "Exec:", cmd.String())
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	bf := &bytes.Buffer{}
	cmd.Stdout = bf
	err = cmd.Run()
	if err != nil {
		return err
	}
	if bf.Len() == 0 {
		log.Println(idStr, "Dir:", dir, "Empty Result")
		return nil
	}
	outFilePath := filepath.Join(*outDir, strings.ReplaceAll(mod.Module.Mod.Path, "/", "_")+".jsonl")
	err = os.WriteFile(outFilePath, bf.Bytes(), 0644)
	return err
}
