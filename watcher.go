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

	log "github.com/Sirupsen/logrus"
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

// addDirRecursively adds file to watcher under given root directory recursively.
func (f *fswatch) addDirRecursively(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			base := info.Name()
			if base != "." && strings.HasPrefix(base, ".") {
				log.Debug("Skip:", path)
				return filepath.SkipDir
			}
			if err := f.watcher.Add(path); err != nil {
				return err
			}
			log.Debug("Add:", path)
		}
		return nil
	})
}

// handleEvent handles fsnotify events and OS signal.
func (f *fswatch) handleEvent() {
	for {
		select {
		case event := <-f.watcher.Events:
			// ignore the event if modified file name is not matched with filter
			if f.Filter != "" {
				match, err := regexp.MatchString(f.Filter, event.Name)
				if err != nil {
					log.Warn(err, event.Name)
					break
				}
				if !match {
					log.Debug("Ignore file changed:", event.Name)
					break
				}
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Info("Modified:", event.Name)
				f.localSig <- "Modified"
			}
			if event.Op&fsnotify.Create == fsnotify.Create {
				log.Info("Created:", event.Name)
				f.localSig <- "Created"
			}
			if event.Op&fsnotify.Remove == fsnotify.Remove {
				log.Info("Removed:", event.Name)
				f.localSig <- "Removed"
			}
		case err := <-f.watcher.Errors:
			log.Error(err)
		case <-f.osSignal:
			fmt.Println()
			f.localSig <- "Interrupt"
		}
	}
}

func (f *fswatch) Watch() (err error) {
	if f.watcher, err = fsnotify.NewWatcher(); err != nil {
		log.Fatal(err)
	}
	defer f.watcher.Close()

	for _, path := range f.Paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Fatal("no such file or directory:", path)
		}
		if err = f.addDirRecursively(path); err != nil {
			log.Fatal(err)
		}
	}

	f.osSignal = make(chan os.Signal)
	signal.Notify(f.osSignal, syscall.SIGINT)

	f.localSig = make(chan string)

	go func() {
		for {
			log.Info("Run:", f.Command)
			c := exec.Command(f.Command[0], f.Command[1:]...)
			c.Stdout = os.Stdout
			c.Stderr = os.Stdout

			c.SysProcAttr = &syscall.SysProcAttr{}
			c.SysProcAttr.Setpgid = true
			if err := c.Start(); err != nil {
				log.Fatal(err)
			}

			done := make(chan error, 1)
			go func() {
				done <- c.Wait()
			}()
			select {
			case msg := <-f.localSig:
				if err := c.Process.Kill(); err != nil {
					log.Fatal("failed to kill:", err)
				}
				<-done
				log.Info("Stop")
				if msg == "Interrupt" {
					log.Info("Exit")
					os.Exit(1)
				}
				goto SKIP_WAITING
			case err := <-done:
				if err != nil {
					log.Error("process done with error =", err)
				}
			}
			log.Info("Wait for signal...")
			if msg := <-f.localSig; msg == "Interrupt" {
				log.Info("Exit")
				os.Exit(1)
			}
		SKIP_WAITING:
			time.Sleep(1)
		}
	}()

	f.handleEvent()
	return nil
}
