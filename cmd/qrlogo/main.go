package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		if e, ok := err.(*exitError); ok {
			os.Exit(e.code)
		}
		os.Exit(1)
	}
}
