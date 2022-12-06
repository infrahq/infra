package main

import (
	"fmt"
	"os"
	"strings"
)

// echo and printf do not always work the same way on macos and linux. This
// command gives us a reliable way of echoing args in a test.
func main() {
	for i, arg := range os.Args[1:] {
		if i != 0 {
			fmt.Print(" ")
		}
		if strings.ContainsAny(arg, " \t\n*\"") {
			fmt.Printf("'%v'", arg)
			continue
		}
		fmt.Print(arg)
	}
	fmt.Println()
}
