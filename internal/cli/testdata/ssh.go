package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	switch os.Args[len(os.Args)-1] {
	case "fixture-fail":
		fmt.Fprintln(os.Stdout, "partial output")
		fmt.Fprintln(os.Stderr, "permission denied")
		os.Exit(7)
	case "fixture-sleep":
		time.Sleep(5 * time.Second)
	case "fixture-child":
		child := exec.Command(os.Args[0], "fixture-sleep")
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
		if err := child.Start(); err != nil {
			os.Exit(9)
		}
		if err := os.WriteFile(os.Getenv("HOVA_TEST_CHILD_PID"), []byte(fmt.Sprint(child.Process.Pid)), 0600); err != nil {
			os.Exit(10)
		}
		time.Sleep(5 * time.Second)
	default:
		fmt.Fprintln(os.Stdout, "remote output")
	}
}
