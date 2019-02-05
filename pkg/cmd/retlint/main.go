// Copyright 2019 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	l := &RetLint{}

	var setExitStatus bool
	root := cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			dirty, err := l.Execute()
			if err != nil {
				return err
			}
			if len(dirty) == 0 {
				return nil
			}
			for _, d := range dirty {
				fmt.Println(d)
			}
			if setExitStatus {
				return errors.New("dirty")
			}
			return nil
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.Flags().StringSliceVarP(&l.AllowedNames, "allow", "a", nil, "the allowed types to return")
	root.Flags().StringVarP(&l.Dir, "dir", "d", ".", "the working directory to use")
	root.Flags().StringSliceVarP(&l.Packages, "pkg", "p", []string{"./..."}, "the package(s) to lint")
	root.Flags().BoolVar(&setExitStatus, "set_exit_status", false, "set exit status to 1 if any functions are dirty")
	root.Flags().StringVarP(&l.TargetName, "target", "t", "", "the target interface type")

	if err := root.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}
