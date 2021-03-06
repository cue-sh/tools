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

package ci

import (
	"path"

	"encoding/yaml"

	"tool/exec"
	"tool/file"
	"tool/os"
)

// genworkflows regenerates the GitHub workflow Yaml definitions.
//
// See internal/ci/gen.go for details on how this step fits into the sequence
// of generating our CI workflow definitions, and updating various txtar tests
// with files from that process.
//
// Until we have a resolution for cuelang.org/issue/704 and
// cuelang.org/issue/708 this must be run from the internal/ci package. At
// which point we can switch to using _#modroot.
//
// This also explains why the ../../ relative path specification below appear
// wrong in the context of the containing directory internal/ci/vendor.
command: genworkflows: {
	goos: _#goos

	for w in workflows {
		"\(w.file)": file.Create & {
			_dir:     path.FromSlash("../../.github/workflows", path.Unix)
			filename: path.Join([_dir, w.file], goos.GOOS)
			contents: """
						 # Generated by internal/ci/ci_tool.cue; do not edit

						 \(yaml.Marshal(w.schema))
						 """
		}
	}
}

// updateTxtarTests ensures certain txtar tests are updated with the
// relevant files that make up the process of generating our CI
// workflows.
//
// See internal/ci/gen.go for details on how this step fits into the sequence
// of generating our CI workflow definitions, and updating various txtar tests
// with files from that process.
//
// Until we have a resolution for cuelang.org/issue/704 and
// cuelang.org/issue/708 this must be run from the internal/ci package. At
// which point we can switch to using _#modroot.
//
// This also explains why the ../../ relative path specification below appear
// wrong in the context of the containing directory internal/ci/vendor.
command: updateTxtarTests: {
	goos: _#goos

	readJSONSchema: file.Read & {
		_path:    path.FromSlash("../../cue.mod/pkg/github.com/SchemaStore/schemastore/src/schemas/json/github-workflow.cue", path.Unix)
		filename: path.Join([_path], goos.GOOS)
		contents: string
	}
	cueDefInternalCI: exec.Run & {
		cmd:    "go run cuelang.org/go/cmd/cue def cuelang.org/go/internal/ci"
		stdout: string
	}
	// updateEvalTxtarTest updates the cue/testdata/eval testscript which exercises
	// the evaluation of the workflows defined in internal/ci (which by definition
	// means resolving and using the vendored GitHub Workflow schema)
	updateEvalTxtarTest: {
		_relpath: path.FromSlash("../../cue/testdata/eval/github.txtar", path.Unix)
		_path:    path.Join([_relpath], goos.GOOS)

		githubSchema: exec.Run & {
			stdin: readJSONSchema.contents
			cmd:   "go run cuelang.org/go/internal/ci/updateTxtar - \(_path) cue.mod/pkg/github.com/SchemaStore/schemastore/src/schemas/json/github-workflow.cue"
		}
		defWorkflows: exec.Run & {
			$after: githubSchema
			stdin:  cueDefInternalCI.stdout
			cmd:    "go run cuelang.org/go/internal/ci/updateTxtar - \(_path) workflows.cue"
		}
	}
	// When we have a solution for cuelang.org/issue/709 we can make this a
	// file.Glob
	readToolsFile: file.Read & {
		filename: "ci_tool.cue"
		contents: string
	}
	updateCmdCueCmdTxtarTest: {
		_relpath: path.FromSlash("../../cmd/cue/cmd/testdata/script/cmd_github.txt", path.Unix)
		_path:    path.Join([_relpath], goos.GOOS)

		githubSchema: exec.Run & {
			stdin: readJSONSchema.contents
			cmd:   "go run cuelang.org/go/internal/ci/updateTxtar - \(_path) cue.mod/pkg/github.com/SchemaStore/schemastore/src/schemas/json/github-workflow.cue"
		}
		defWorkflows: exec.Run & {
			$after: githubSchema
			stdin:  cueDefInternalCI.stdout
			cmd:    "go run cuelang.org/go/internal/ci/updateTxtar - \(_path) internal/ci/workflows.cue"
		}
		toolsFile: exec.Run & {
			stdin: readToolsFile.contents
			cmd:   "go run cuelang.org/go/internal/ci/updateTxtar - \(_path) internal/ci/\(readToolsFile.filename)"
		}
	}
}

// _#modroot is a common helper to get the module root
//
// TODO: use once we have a solution to cuelang.org/issue/704.
// This will then allow us to remove the use of .. below.
_#modroot: exec.Run & {
	cmd:    "go list -m -f {{.Dir}}"
	stdout: string
}

// Until we have the ability to inject contextual information
// we need to pass in GOOS explicitly. Either by environment
// variable (which we get for free when this is used via go generate)
// or via a tag in the case you want to manually run the CUE
// command.
_#goos: os.Getenv & {
	GOOS: *path.Unix | string @tag(os)
}
