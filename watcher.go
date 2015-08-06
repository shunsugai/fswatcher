package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	fsnotify "gopkg.in/fsnotify.v1"
)

type fswatch struct {
	Command  []string
	Paths    []string
	Filter   string
	osSignal chan os.Signal
	localSig chan string
	watcher  *fsnotify.Watcher
}

func (f *fswatch) addDirRecursively(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			base := info.Name()
			if base != "." && strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			if err := f.watcher.Add(path); err != nil {
				return err
			}
		}
		return nil
	})
}

func (f *fswatch) Watch() (err error) {
	if f.watcher, err = fsnotify.NewWatcher(); err != nil {
		cfatal(err)
	}
	defer f.watcher.Close()

	for _, path := range f.Paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			cfatal("no such file or directory:", path)
		}
		if err = f.addDirRecursively(path); err != nil {
			cfatal(err)
		}
	}

	f.osSignal = make(chan os.Signal)
	signal.Notify(f.osSignal, syscall.SIGINT)

	f.localSig = make(chan string)

	go func() {
		for {
			cprintln("Start command:", f.Command)
			c := exec.Command(f.Command[0], f.Command[1:]...)
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
			case msg := <-f.localSig:
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
			if msg := <-f.localSig; msg == "Interrupt" {
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
		case event := <-f.watcher.Events:
			// ignore the event if modified file name is not matched with filter
			if f.Filter != "" {
				match, err := regexp.MatchString(f.Filter, event.Name)
				if err != nil {
					break
				}
				if !match {
					break
				}
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				fmt.Println()
				cprintln("Modified file:", event.Name)
				f.localSig <- "Modified"
			}
			if event.Op&fsnotify.Create == fsnotify.Create {
				fmt.Println()
				cprintln("Created file:", event.Name)
				f.localSig <- "Created"
			}
			if event.Op&fsnotify.Remove == fsnotify.Remove {
				fmt.Println()
				cprintln("Removed file:", event.Name)
				f.localSig <- "Removed"
			}
		case err := <-f.watcher.Errors:
			cprintln("ERROR:", err)
		case <-f.osSignal:
			fmt.Println()
			f.localSig <- "Interrupt"
		}
	}
}
