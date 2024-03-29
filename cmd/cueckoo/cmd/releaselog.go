// Copyright 2023 The CUE Authors
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

package cmd

import (
	"fmt"
	"strings"

	"github.com/google/go-github/v53/github"
	"github.com/spf13/cobra"
)

// newUnityCmd creates a new unity command
func newReleaselogCmd(c *Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "releaselog",
		Short: "create a GitHub release log",
		Long: `
Usage of releaselog:

	releaselog RANGE_START RANGE_END

releaselog generates a bullet list of commits similar to the GitHub change log
that is automatically created for a release in a repository that uses pull
requests. Because the CUE repository does not use PRs, the automatic change log
refuses to generate.

RANGE_START and RANGE_END are both required. The arguments are interpreted in a
similar way to:

    git log $RANGE_START..$RANGE_END

Like git log, commits are in reverse chronological order.
`,
		RunE: mkRunE(c, releaseLog),
	}
	return cmd
}

func releaseLog(cmd *Command, args []string) error {
	cmd.Flags()

	if len(args) != 2 {
		return fmt.Errorf("expected exactly two args which will be interpreted like git log $1..$2, like: v0.8.0-alpha.1 master")
	}
	fromRef, toRef := args[0], args[1]

	cfg, err := loadConfig(cmd.Context())
	if err != nil {
		return err
	}

	var commits []*github.RepositoryCommit
	opts := &github.ListOptions{
		Page: 1,
	}

	// Gather commits and authors
	for {
		res, resp, err := cfg.githubClient.Repositories.CompareCommits(cmd.Context(), cfg.githubOwner, cfg.githubRepo, fromRef, toRef, opts)
		// Check for any errors
		if err != nil {
			return fmt.Errorf("failed to compare commits: %w", err)
		}

		// Extract the commits
		commits = append(commits, res.Commits...)

		// Break if done. For some reason, when there is just one page of results
		// resp.LastPage is 0. Who would have thought?!
		if resp.LastPage <= opts.Page {
			break
		}
		opts.Page++
	}

	fmt.Printf("<details>\n\n<summary><b>Full list of changes since %s</b></summary>\n\n", fromRef)
	for i := len(commits) - 1; i >= 0; i-- {
		commit := commits[i]
		msg := commit.Commit.GetMessage()
		author := commit.GetAuthor().GetLogin()
		summary, _, _ := strings.Cut(msg, "\n")
		fmt.Printf("* %s by @%s in %s\n", summary, author, commit.GetSHA())
	}
	fmt.Printf("\n</details>\n")

	return nil
}
