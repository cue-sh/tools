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

package cmd

import (
	"fmt"

	"github.com/google/go-github/v53/github"
	"github.com/spf13/cobra"
)

const (
	flagRunTrybotNoUnity flagName = "nounity"
	flagForce            flagName = "force"
)

// newRuntrybotCmd creates a new runtrybot command
func newRuntrybotCmd(c *Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "runtrybot",
		Short: "run the CUE trybot on the given CL",
		Long: `
Usage of runtrybot:

	runtrybot [--nounity] [ARGS...]

Triggers trybot and unity runs for its arguments.

When run with no arguments, runtrybot derives a revision and change ID for each
pending commit in the current branch. If multiple pending commits are found,
you must either specify which commits or CLs to run, or specify HEAD to run the
trybots for all of them.

runtrybot needs your GitHub username and a personal acccess token
with the "repo" scope. You can configure them via your git credential helper,
or by setting the GITHUB_USER and GITHUB_PAT environment variables.
Note that the personal access token should be "classic"; GitHub's new
fine-grained tokens are still in beta and haven't been tested to work here.

If the --nounity flag is provided, only a trybot run is triggered.
`,
		RunE: mkRunE(c, runtrybotDef),
	}
	cmd.Flags().Bool(string(flagRunTrybotNoUnity), false, "do not simultaenously trigger unity build")
	cmd.Flags().BoolP(string(flagForce), string(flagForce[0]), false, "force the trybots to run, ignoring any results")
	return cmd
}

func runtrybotDef(cmd *Command, args []string) error {
	cfg, err := loadConfig(cmd.Context())
	if err != nil {
		return err
	}
	r := newCLTrigger(cmd, cfg, func(payload repositoryDispatchPayload) error {
		trybotPayload := payload
		trybotPayload.Type = string(eventTypeTrybot)
		p, err := buildTryBotPayload(trybotPayload)
		if err != nil {
			return err
		}
		if err := cfg.triggerRepositoryDispatch(cfg.githubOwner, cfg.githubRepo, p); err != nil {
			return err
		}
		if cfg.unityRepo != "" && !flagRunTrybotNoUnity.Bool(cmd) {
			unityPayload := payload
			unityPayload.Type = string(eventTypeUnity)
			p, err := buildUnityPayloadFromCLTrigger(unityPayload)
			if err != nil {
				return err
			}
			if err := cfg.triggerRepositoryDispatch(cfg.unityOwner, cfg.unityRepo, p); err != nil {
				return err
			}
		}
		return nil
	})
	return r.run()
}

func buildTryBotPayload(payload repositoryDispatchPayload) (github.DispatchRequestOptions, error) {
	msg := fmt.Sprintf("trybot run for %v", payload.Ref)
	return buildDispatchPayload(msg, payload)
}
