# Go-Mod-Download

## 1 Install
```bash
go install github.com/fsgo/gomodule/cmd/go-mod-download@master
```

## 2 Usage
```bash
go-mod-download -f modules.txt -d "./modules_download"
```

modules.txt :
```text
golang.org/x/sync
golang.org/x/text
golang.org/x/time
```