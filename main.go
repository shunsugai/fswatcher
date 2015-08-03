package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	fsnotify "gopkg.in/fsnotify.v1"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

const appName = "fswatcher"

func addDirRecursively(root string, w *fsnotify.Watcher, filter string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			base := info.Name()
			if base != "." && strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			if err := w.Add(path); err != nil {
				return err
			}
		} else {
			match, err := regexp.MatchString(filter, path)
			if err != nil {
				return err
			}
			if !match {
				cprintln("Ignore:", path)
				if err := w.Remove(path); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func doWatch(paths cli.Args, cmd []string, filter string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		cfatal(err)
	}
	defer watcher.Close()

	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			cfatal("no such file or directory:", path)
		}
		if err = addDirRecursively(path, watcher, filter); err != nil {
			cfatal(err)
		}
	}

	osSignal := make(chan os.Signal)
	signal.Notify(osSignal, syscall.SIGINT)

	localSig := make(chan string)

	go func() {
		for {
			cprintln("Start command:", cmd)
			c := exec.Command(cmd[0], cmd[1:]...)
			c.Stdout = os.Stdout
			c.Stderr = os.Stdout

			c.SysProcAttr = &syscall.SysProcAttr{}
			c.SysProcAttr.Setpgid = true
			err := c.Start()
			if err != nil {
				cfatal(err)
			}

			done := make(chan error, 1)
			go func() {
				done <- c.Wait()
			}()
			select {
			case msg := <-localSig:
				if err := c.Process.Kill(); err != nil {
					cfatal("failed to kill:", err)
				}
				<-done
				cprintln("Stop command")
				if msg == "Interrupt" {
					cprintln("Exit")
					os.Exit(1)
				}
				goto SKIP_WAITING
			case err := <-done:
				if err != nil {
					cfatal("process done with error =", err)
				}
			}
			cprintln("Wait for signal...")
			if msg := <-localSig; msg == "Interrupt" {
				cprintln("Exit")
				os.Exit(1)
			}
		SKIP_WAITING:
			time.Sleep(1)
		}
	}()

	// handle event
	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				fmt.Println()
				cprintln("Modified file:", event.Name)
				localSig <- "Modified"
			}
			if event.Op&fsnotify.Create == fsnotify.Create {
				fmt.Println()
				cprintln("Created file:", event.Name)
				localSig <- "Created"
			}
			if event.Op&fsnotify.Remove == fsnotify.Remove {
				fmt.Println()
				cprintln("Removed file:", event.Name)
				localSig <- "Removed"
			}
		case err := <-watcher.Errors:
			cprintln("ERROR:", err)
		case <-osSignal:
			fmt.Println()
			localSig <- "Interrupt"
		}
	}
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
	app.Name = appName
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
		cprintln("Now watching at:")
		for _, arg := range c.Args() {
			abs, err := filepath.Abs(arg)
			if err != nil {
				cprintln("ERROR: failed to convert to absolute path:", arg)
			}
			cprintln("\t", abs)
		}
		cmds := strings.Split(c.String("exec"), " ")
		filter := c.String("includefilter")
		doWatch(c.Args(), cmds, filter)
	}
	app.Run(os.Args)
}
