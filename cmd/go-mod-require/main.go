// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/4/12

package main

import (
	"bytes"
	"context"
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"golang.org/x/mod/modfile"
)

const goRecoverCmd = "go-recover -debug=v -test=false ./..."

var run = flag.String("exec", "", "exec cmd in required module dir")
var gr = flag.Bool("gr", false, "exec: "+goRecoverCmd)

func main() {
	flag.Parse()

	bf, err := os.ReadFile("go.mod")
	if err != nil {
		log.Fatalln("read go.mod failed:", err)
	}
	mf, err := modfile.Parse("go.mod", bf, nil)
	if err != nil {
		log.Fatalln("parser go.mod failed:", err)
	}

	total := len(mf.Require)
	log.Println("total:", total)

	for i, r := range mf.Require {
		fp := filepath.Join(goModCacheDir, cacheDirName(r.Mod.String()))
		log.Println(color.CyanString("Module[%d/%d]: %s  Dir: %s", i+1, total, r.Mod.String(), fp))
		execAt(fp)
	}
}

func cacheDirName(fp string) string {
	var bf bytes.Buffer
	for i := 0; i < len(fp); i++ {
		c := string(fp[i])
		lc := strings.ToLower(c)
		if c != lc {
			bf.WriteString("!")
		}
		bf.WriteString(lc)
	}
	return bf.String()
}

func getExec() string {
	if *gr {
		return goRecoverCmd
	}
	return strings.TrimSpace(*run)
}

func execAt(dir string) {
	str := getExec()
	if str == "" {
		return
	}
	rr := strings.Fields(str)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, rr[0], rr[1:]...)
	log.Println("exec:", cmd.String())
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Dir = dir
	err := cmd.Run()
	if err != nil {
		log.Println(color.RedString("exec failed: %v", err))
	} else {
		log.Println(color.GreenString("exec success"))
	}
}

var goModCacheDir string

func init() {
	cmd := exec.Command("go", "env", "GOMODCACHE")
	out, err := cmd.Output()
	if err != nil {
		log.Fatalln("get GOMODCACHE failed:", err)
	}
	goModCacheDir = strings.TrimSpace(string(out))
}
