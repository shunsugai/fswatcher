package main

import (
	"os"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/mattn/go-colorable"
)

const (
	APP_NAME = "fswatcher"
)

var (
	fsw = &fswatch{
		Command: []string{"echo", "Hello, World"},
		Paths:   []string{"."},
		Filter:  "",
	}
)

func main() {
	app := cli.NewApp()
	app.Name = APP_NAME
	app.Usage = "Executes command when file or directories are modified"
	app.Version = "0.0.1"
	app.Author = "Shun Sugai"
	app.Email = "sugaishun@gmail.com"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "exec, e",
			Usage: "command to execute",
		},
		cli.StringFlag{
			Name:  "includefilter, i",
			Usage: "filter to include. e.g. .(go|rb|java)",
		},
	}
	app.Action = func(c *cli.Context) {
		if len(c.Args()) < 1 {
			cli.ShowAppHelp(c)
			os.Exit(1)
		}
		fsw.Command = strings.Split(c.String("exec"), " ")
		fsw.Paths = []string(c.Args())
		fsw.Filter = c.String("includefilter")
		for _, path := range fsw.Paths {
			abs, err := filepath.Abs(path)
			if err != nil {
				log.Warn("failed to convert to absolute path:", path)
			}
			log.Info("Now watching at:", abs)
		}
		fsw.Watch()
	}
	app.Run(os.Args)
}

func init() {
	log.SetOutput(colorable.NewColorableStdout())
	log.SetLevel(log.InfoLevel)
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
