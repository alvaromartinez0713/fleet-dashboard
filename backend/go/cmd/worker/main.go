package main

import "os"

func main() {
	exitCode := Run()
	os.Exit(exitCode)
}
