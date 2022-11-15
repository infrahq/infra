// Based on https://github.com/spf13/cobra/blob/master/doc/md_docs.go
// Copyright 2015 Red Hat Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

func printOptions(buf *bytes.Buffer, cmd *cobra.Command, name string) error {
	flags := cmd.NonInheritedFlags()
	flags.SetOutput(buf)

	if flags.HasAvailableFlags() {
		buf.WriteString("#### Options\n\n```console\n")
		flags.PrintDefaults()
		buf.WriteString("```\n\n")
	}

	parentFlags := cmd.InheritedFlags()
	parentFlags.SetOutput(buf)

	if parentFlags.HasAvailableFlags() {
		buf.WriteString("**Additional options**\n\n```console\n")
		parentFlags.PrintDefaults()
		buf.WriteString("```\n")
	}

	return nil
}

// GenMarkdownCustom creates custom markdown output.
func GenMarkdown(cmd *cobra.Command, w io.Writer) error {
	cmd.InitDefaultHelpCmd()
	cmd.InitDefaultHelpFlag()

	buf := new(bytes.Buffer)
	name := cmd.CommandPath()

	buf.WriteString("### `" + name + "`\n\n")
	buf.WriteString(cmd.Short + "\n\n")

	if len(cmd.Long) > 0 {
		buf.WriteString("#### Description\n\n")
		buf.WriteString(cmd.Long + "\n\n")
	}

	if cmd.Runnable() {
		buf.WriteString(fmt.Sprintf("```bash\n%s\n```\n\n", cmd.UseLine()))
	}

	if len(cmd.Example) > 0 {
		buf.WriteString("#### Examples\n\n")
		buf.WriteString(fmt.Sprintf("```bash\n%s\n```\n\n", cmd.Example))
	}

	if err := printOptions(buf, cmd, name); err != nil {
		return err
	}

	_, err := buf.WriteTo(w)

	return err
}

func GenMarkdownToc(cmd *cobra.Command, f io.Writer) error {
	if len(cmd.Commands()) == 0 {
		if _, err := io.WriteString(f, "* ["+cmd.CommandPath()+"](#"+strings.ToLower(strings.ReplaceAll(cmd.CommandPath(), " ", "-"))+")\n"); err != nil {
			return err
		}
	}

	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}

		if err := GenMarkdownToc(c, f); err != nil {
			return err
		}
	}

	return nil
}

func GenMarkdownTree(cmd *cobra.Command, f io.Writer) error {
	if len(cmd.Commands()) == 0 {
		return GenMarkdown(cmd, f)
	}

	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}

		if err := GenMarkdownTree(c, f); err != nil {
			return err
		}
	}

	return nil
}

func GenMarkdownFile(cmd *cobra.Command, f io.Writer) error {
	if _, err := io.WriteString(f, `---
title: CLI Reference
---
`); err != nil {
		return err
	}

	if _, err := io.WriteString(f, "# CLI Reference"); err != nil {
		return err
	}

	if _, err := io.WriteString(f, "\n\n"); err != nil {
		return err
	}

	if _, err := io.WriteString(f, "## Commands"); err != nil {
		return err
	}

	if _, err := io.WriteString(f, "\n\n"); err != nil {
		return err
	}

	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}

		if err := GenMarkdownTree(c, f); err != nil {
			return err
		}
	}

	return nil
}
