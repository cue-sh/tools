// Copyright 2021 The CUE Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// ****************************************************************
// This file contains definitions which are common to the approach used for
// cue/cmd/cmd. i.e. they are copy-pasted from cmd/cue
// ****************************************************************

package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

type panicError struct {
	Err error
}

func exit() {
	panic(panicError{errPrintedError})
}

func recoverError(err *error) {
	switch e := recover().(type) {
	case nil:
	case panicError:
		*err = e.Err
	default:
		panic(e)
	}
	// We use panic to escape, instead of os.Exit
}

type Command struct {
	*cobra.Command
	root   *cobra.Command
	hasErr bool
}

func (c *Command) Run(ctx context.Context) (err error) {
	// Three categories of commands:
	// - normal
	// - user defined
	// - help
	// For the latter two, we need to use the default loading.
	defer recoverError(&err)

	if err := c.root.Execute(); err != nil {
		return err
	}
	if c.hasErr {
		return errPrintedError
	}
	return nil
}

func (c *Command) SetOutput(w io.Writer) {
	c.root.SetOut(w)
}

func (c *Command) SetInput(r io.Reader) {
	c.root.SetIn(r)
}

func (c *Command) Stderr() io.Writer {
	return (*errWriter)(c)
}

type flagName string

func (f flagName) Bool(cmd *Command) bool {
	v, _ := cmd.Flags().GetBool(string(f))
	return v
}

func (f flagName) String(cmd *Command) string {
	v, _ := cmd.Flags().GetString(string(f))
	return v
}

func (f flagName) StringArray(cmd *Command) []string {
	v, _ := cmd.Flags().GetStringArray(string(f))
	return v
}

var errPrintedError = errors.New("terminating because of errors")

func mkRunE(c *Command, f runFunction) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		c.Command = cmd
		err := f(c, args)
		if err != nil {
			exitOnErr(c, err, true)
		}
		return err
	}
}

type errWriter Command

func (w *errWriter) Write(b []byte) (int, error) {
	c := (*Command)(w)
	c.hasErr = true
	return c.Command.OutOrStderr().Write(b)
}

func exitOnErr(cmd *Command, err error, fatal bool) {
	if err == nil {
		return
	}

	fmt.Fprintln(os.Stderr, err)
	if fatal {
		exit()
	}
}

type runFunction func(cmd *Command, args []string) error
