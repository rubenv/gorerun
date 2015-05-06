// gorerun is equivalent to "go run".
//
// The only difference is that every time you send it a SIGHUP, it will
// restart the process. This is very useful while developing.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

var cmd *exec.Cmd

func main() {
	restart := true

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGHUP)

		// Trigger a restart whenever a SIGHUP comes in
		for {
			<-c
			restart = true
			cmd.Process.Kill()
		}
	}()

	for restart {
		restart = false

		// Start process
		args := append([]string{"run"}, os.Args[1:]...)
		cmd = exec.Command("go", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()

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
