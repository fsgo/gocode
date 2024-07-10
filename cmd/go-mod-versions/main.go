// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/8/11

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/fsgo/cmdutil"
)

var input = flag.String("in", "modules.txt", "input file")
var out = flag.String("out", "module_version.toml", "output file")

var conc = flag.Int("c", 10, "Number of multiple task to make at a time")
var goName = flag.String("go", "go", "go command")

func main() {
	flag.Parse()
	bf, err := os.ReadFile(*input)
	if err != nil {
		log.Fatalln(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}

	cacheDir := filepath.Join(wd, "gocache")
	os.Setenv("GOMODCACHE", cacheDir)
	os.Setenv("GOCACHE", cacheDir)

	defer func() {
		cmd := exec.Command(*goName, "clean", "-modcache")
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		cmd.Run()
	}()

	lines := strings.Split(string(bf), "\n")
	sort.Strings(lines)

	his := make(map[string]bool, len(lines))

	_ = os.Remove(*out + ".tmp")

	wg := &cmdutil.WorkerGroup{
		Max: *conc,
	}

	for idx := 0; idx < len(lines); idx++ {
		line := strings.TrimSpace(lines[idx])
		if len(line) == 0 || his[line] {
			continue
		}
		his[line] = true

		wg.Run(func() {
			dealOne(idx, len(lines), line)
		})
	}
	wg.Wait()
	os.Rename(*out+".tmp", *out)
}

func dealOne(idx int, total int, name string) {
	prefix := fmt.Sprintf("[%d/%d] %-60s ", idx, total, name)
	logger := log.New(os.Stderr, prefix, 0)
	info, err := getLastVersion(logger, name)
	if err != nil {
		return
	}
	info.ID = idx
	logger.SetPrefix(prefix)

	logger.Println(color.CyanString(info.String()))

	file, err := os.OpenFile(*out+".tmp", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logger.Println(err)
	}
	defer file.Close()
	file.WriteString(info.TomlString() + "\n")
}

func getLastVersion(logger *log.Logger, path string) (v *modInfo, err error) {
	lp := logger.Prefix()
	for i := 0; i < 3; i++ {
		logger.SetPrefix(lp + fmt.Sprintf(" [try %d/%d] ", i+1, 3))
		v, err = getVersion(logger, path)
		if err == nil {
			break
		}
		logger.Println("failed:", err)
	}
	return v, err
}

func getVersion(logger *log.Logger, path string) (*modInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "list", "-m", "-json", path+"@latest")
	logger.Println(cmd.String())
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		return nil, err
	}
	var info *modInfo
	if err = json.NewDecoder(stdout).Decode(&info); err != nil {
		return nil, err
	}
	if err = cmd.Wait(); err != nil {
		return nil, err
	}
	return info, nil
}

type modInfo struct {
	ID      int
	Path    string
	Version string
	Time    time.Time
}

func (mi *modInfo) String() string {
	bf, _ := json.Marshal(mi)
	return string(bf)
}

func (mi *modInfo) TomlString() string {
	rd := mi.Time.Format("2006-01-02")
	tpl := `[[Modules]]
ID = ` + strconv.Itoa(mi.ID) + `
Path = "` + mi.Path + `"
LastVersion = "` + mi.Version + `"
ReleaseDay = "` + rd + `"
`
	return tpl
}
