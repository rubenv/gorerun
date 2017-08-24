// gorerun is equivalent to "go run".
//
// The only difference is that every time you send it a SIGHUP, it will
// restart the process. This is very useful while developing.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var cmd *exec.Cmd

func main() {
	killed := false
	sleep := 10 * time.Second

	flags := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flags.Usage = func() {}
	pkg := flags.String("pkg", "", "Package to install (speeds up reruns)")
	flags.Parse(os.Args[1:])

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGHUP)

		// Trigger a restart whenever a SIGHUP comes in
		for {
			<-c
			killed = true
			pgid, err := syscall.Getpgid(cmd.Process.Pid)
			if err == nil {
				syscall.Kill(-pgid, syscall.SIGTERM)
			}
		}
	}()

	for {
		killed = false

		var err error

		// Check if the entry point exists
		entry := ""
		for _, s := range flags.Args() {
			if strings.HasSuffix(s, ".go") {
				entry = s
			}
		}
		if entry == "" {
			err = errors.New("No entry-point specified!")
		} else {
			_, err = os.Stat(entry)
			if err != nil && os.IsNotExist(err) {
				err = fmt.Errorf("Not found: %s, did you forget to 'go get' the required package?", entry)
			}
		}

		if pkg != nil && *pkg != "" {
			// Pre-compile package
			cmd = exec.Command("go", "install", *pkg)
			err = cmd.Run()
		}

		if err == nil {
			// Start process
			args := append([]string{"run"}, flags.Args()...)
			cmd = exec.Command("go", args...)
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
		}

		if !killed {
			if err != nil {
				if e, ok := err.(*exec.ExitError); ok {
					if status, ok := e.Sys().(syscall.WaitStatus); ok {
						log.Printf("Exited with status %d", status.ExitStatus())
					}
				} else {
					log.Println(err)
				}
			}
			log.Println("Sleeping %s before restarting", sleep)
			time.Sleep(sleep)
		}
	}
}
