package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"gopkg.in/fsnotify.v1"
	"log"
	"os"
	"os/exec"
	"strings"
)

func execCommand(cmd []string) {
	out, err := exec.Command(cmd[0], cmd[1:]...).Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s", out)
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
		err = watcher.Add(path)
		if err != nil {
			log.Fatal(err)
		}
	}
	<-done
}

func initialize() {
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
	initialize()
	app := cli.NewApp()
	app.Name = "watcher"
	app.Usage = "Executes command when file or directories are modified"
	app.Version = "0.0.1"
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
