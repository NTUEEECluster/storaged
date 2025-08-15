package main

import (
	"fmt"
	"os"
)

func main() {
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error running program:", err)
	}
}

func run() error {
	return nil
}
