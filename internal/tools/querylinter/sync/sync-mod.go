package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"golang.org/x/mod/modfile"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(_ []string) error {
	binary, err := exec.LookPath("golangci-lint")
	if err != nil {
		return fmt.Errorf("lookup golangci-lint in PATH: %w", err)
	}
	fmt.Println("Found", binary)

	out, err := execCmd("go", "version", "-m", binary)
	if err != nil {
		return fmt.Errorf("version for golangci-lint: %w", err)
	}

	target, err := parseTargetVersions(out)
	if err != nil {
		return fmt.Errorf("parse output from 'go version -m': %w", err)
	}
	if target.GoVersion != runtime.Version() {
		return fmt.Errorf("Go version does not match runtime version %v", runtime.Version())
	}

	goModFile, err := readGoMod()
	if err != nil {
		return fmt.Errorf("read go.mod: %w", err)
	}
	for _, require := range goModFile.Require {
		targetVersion, ok := target.Modules[require.Mod.Path]
		if !ok || targetVersion == require.Mod.Version {
			continue
		}
		fmt.Printf("UPDATE: module %v needs version %v\n", require.Mod.Path, targetVersion)
		require.Mod.Version = targetVersion
	}
	goModFile.SetRequire(goModFile.Require)

	if err := writeFile(goModFile); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	if _, err := execCmd("go", "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}
	return nil
}

func writeFile(file *modfile.File) error {
	file.Cleanup()

	raw, err := file.Format()
	if err != nil {
		return fmt.Errorf("failed to format: %w", err)
	}

	return os.WriteFile("go.mod", raw, 0644)
}

func readGoMod() (*modfile.File, error) {
	raw, err := os.ReadFile("go.mod")
	if err != nil {
		return nil, err
	}
	return modfile.Parse("go.mod", raw, nil)
}

func execCmd(cmd string, args ...string) (*bytes.Buffer, error) {
	c := exec.Command(cmd, args...)
	stdout := new(bytes.Buffer)
	c.Stdout = stdout
	c.Stderr = os.Stderr
	return stdout, c.Run()
}

func parseTargetVersions(source io.Reader) (targetVersions, error) {
	target := targetVersions{Modules: make(map[string]string)}
	scan := bufio.NewScanner(source)
	if !scan.Scan() {
		return target, fmt.Errorf("no output")
	}
	_, target.GoVersion, _ = strings.Cut(scan.Text(), ": ")
	for scan.Scan() {
		fields := strings.Fields(scan.Text())
		if len(fields) != 4 || fields[0] != "dep" {
			continue
		}
		target.Modules[fields[1]] = fields[2]
	}

	return target, scan.Err()
}

type targetVersions struct {
	GoVersion string
	Modules   map[string]string
}
