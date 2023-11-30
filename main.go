package main

import (
	"os"
	"smithy/cmd"
)

func main() {
	os.Exit(cmd.Run(os.Args[1:]))
}
