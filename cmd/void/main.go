package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func main() {
	if len(os.Args) < 2 {
		// os.Stderr.WriteString("usage: void <command> [args...]\n")
		fmt.Println("Burppppp")
		os.Exit(1)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		fmt.Println("\b\bForced burppppp!")
		os.Exit(0)
	}()

	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		fmt.Println("Burppppp!")
		os.Exit(1)
	}
	fmt.Println("Burppppp")
}
