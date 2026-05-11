//go:build !debug

package main

import (
	"fmt"
	"os"
)

func runStudio() {
	fmt.Println("Studio mode is not available in release builds.")
	fmt.Printf("Build with: go build -tags debug -o %s . && ./%s studio\n", appCommandName, appCommandName)
	os.Exit(1)
}
