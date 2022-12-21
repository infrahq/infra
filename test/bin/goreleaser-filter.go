package main

import (
	"flag"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	var filename string

	fs := flag.NewFlagSet("main", flag.ContinueOnError)
	fs.StringVar(&filename, "filename", ".goreleaser.yml", "Input file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	fh, err := os.Open(filename)
	if err != nil {
		return err
	}

	config := map[string]any{}
	if err := yaml.NewDecoder(fh).Decode(&config); err != nil {
		return err
	}

	builds := config["builds"].([]any)[0].(map[string]any)
	builds["goos"] = []string{"linux"}
	builds["goarch"] = []string{"amd64"}

	delete(config, "scoop")

	return yaml.NewEncoder(os.Stdout).Encode(config)
}
