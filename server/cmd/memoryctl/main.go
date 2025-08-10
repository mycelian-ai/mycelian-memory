package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "This binary has moved to cmd/memoryctl. Please rebuild.")
	os.Exit(2)
}
