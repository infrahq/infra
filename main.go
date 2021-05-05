package main

import "log"

func main() {
	if err := CmdRun(); err != nil {
		log.Fatal(err)
	}
}
