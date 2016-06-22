// gorerun is equivalent to "go run".
//
// The only difference is that every time you send it a SIGHUP, it will
// restart the process. This is very useful while developing.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

var cmd *exec.Cmd

func main() {
	restart := true

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
			restart = true
			pgid, err := syscall.Getpgid(cmd.Process.Pid)
			if err == nil {
				syscall.Kill(-pgid, syscall.SIGTERM)
			}
		}
	}()

	for restart {
		restart = false

		var err error

		if pkg != nil && *pkg != "" {
			// Pre-compile package
			cmd = exec.Command("go", "install", *pkg)
			err = cmd.Run()
		}

		if err == nil {
			// Start process
			args := append([]string{"run"}, flags.Args()...)
			fmt.Println(args)
			cmd = exec.Command("go", args...)
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
		}

		if !restart && err != nil {
			if e, ok := err.(*exec.ExitError); ok {
				if status, ok := e.Sys().(syscall.WaitStatus); ok {
					os.Exit(status.ExitStatus())
				}
			} else {
				fmt.Print(err)
			}
		}
	}
}
