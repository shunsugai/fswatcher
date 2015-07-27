package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	fsnotify "gopkg.in/fsnotify.v1"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func execCommand(cmd []string) {
	out, err := exec.Command(cmd[0], cmd[1:]...).Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s", out)
}

func addDirRecursively(root string, w *fsnotify.Watcher) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			base := info.Name()
			if base != "." && strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			if err := w.Add(path); err != nil {
				return err
			}
		}
		return nil
	})
}

func doWatch(paths cli.Args, cmd []string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file: ", event.Name)
					execCommand(cmd)
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					log.Println("created file: ", event.Name)
					execCommand(cmd)
				}
				if event.Op&fsnotify.Remove == fsnotify.Remove {
					log.Println("removed file: ", event.Name)
					execCommand(cmd)
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	for _, path := range paths {
		if err = addDirRecursively(path, watcher); err != nil {
			log.Fatal(err)
		}
	}
	<-done
}

func init() {
	cli.AppHelpTemplate = `NAME:
   {{.Name}} - {{.Usage}}

USAGE:
   {{.Name}} [options] [path...]

VERSION:
   {{.Version}}{{if or .Author .Email}}

AUTHOR:{{if .Author}}
  {{.Author}}{{if .Email}} - <{{.Email}}>{{end}}{{else}}
  {{.Email}}{{end}}{{end}}

OPTIONS:
   {{range .Flags}}{{.}}
   {{end}}
`
}

func main() {
	app := cli.NewApp()
	app.Name = "mihari"
	app.Usage = "Executes command when file or directories are modified"
	app.Version = "0.0.1"
	app.Author = "Shun Sugai"
	app.Email = "sugaishun@gmail.com"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "exec, e",
			Usage: "command to execute",
		},
	}
	app.Action = func(c *cli.Context) {
		if len(c.Args()) < 1 {
			cli.ShowAppHelp(c)
			os.Exit(1)
		}
		cmds := strings.Split(c.String("exec"), " ")
		doWatch(c.Args(), cmds)
	}
	app.Run(os.Args)
}
