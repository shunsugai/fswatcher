package main

import (
	"fmt"
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

type watcher struct {
	Command  []string
	Paths    []string
	Filter   string
	osSignal chan os.Signal
	localSig chan string
	w        *fsnotify.Watcher
}

func (this *watcher) addDirRecursively(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			base := info.Name()
			if base != "." && strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			if err := this.w.Add(path); err != nil {
				return err
			}
		} else {
			match, err := regexp.MatchString(this.Filter, path)
			if err != nil {
				return err
			}
			if !match {
				cprintln("Ignore:", path)
				if err := this.w.Remove(path); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (this *watcher) Watch() (err error) {
	if this.w, err = fsnotify.NewWatcher(); err != nil {
		cfatal(err)
	}
	defer this.w.Close()

	for _, path := range this.Paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			cfatal("no such file or directory:", path)
		}
		if err = this.addDirRecursively(path); err != nil {
			cfatal(err)
		}
	}

	this.osSignal = make(chan os.Signal)
	signal.Notify(this.osSignal, syscall.SIGINT)

	this.localSig = make(chan string)

	go func() {
		for {
			cprintln("Start command:", this.Command)
			c := exec.Command(this.Command[0], this.Command[1:]...)
			c.Stdout = os.Stdout
			c.Stderr = os.Stdout

			c.SysProcAttr = &syscall.SysProcAttr{}
			c.SysProcAttr.Setpgid = true
			if err := c.Start(); err != nil {
				cfatal(err)
			}

			done := make(chan error, 1)
			go func() {
				done <- c.Wait()
			}()
			select {
			case msg := <-this.localSig:
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
			if msg := <-this.localSig; msg == "Interrupt" {
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
		case event := <-this.w.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				fmt.Println()
				cprintln("Modified file:", event.Name)
				this.localSig <- "Modified"
			}
			if event.Op&fsnotify.Create == fsnotify.Create {
				fmt.Println()
				cprintln("Created file:", event.Name)
				this.localSig <- "Created"
			}
			if event.Op&fsnotify.Remove == fsnotify.Remove {
				fmt.Println()
				cprintln("Removed file:", event.Name)
				this.localSig <- "Removed"
			}
		case err := <-this.w.Errors:
			cprintln("ERROR:", err)
		case <-this.osSignal:
			fmt.Println()
			this.localSig <- "Interrupt"
		}
	}
}
