package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type BlockStep struct {
	Type                   string       `json:"type"`
	Prompt                 *string      `json:"prompt,omitempty"`
	Branches               *interface{} `json:"branches,omitempty"`
	AllowDependencyFailure *bool        `json:"allow_dependency_failure,omitempty"`
	Block                  *string      `json:"block,omitempty"`
	BlockedState           *string      `json:"blocked_state,omitempty"`
	DependsOn              *interface{} `json:"depends_on,omitempty"`
	Fields                 *interface{} `json:"fields,omitempty"`
	Id                     *string      `json:"id,omitempty"`
	Identifier             *string      `json:"identifier,omitempty"`
	If                     *string      `json:"if,omitempty"`
	Key                    *string      `json:"key,omitempty"`
	Label                  *string      `json:"label,omitempty"`
	Name                   *string      `json:"name,omitempty"`
}

type SimplePipeline struct {
	Steps []BlockStep `json:"steps"`
}

func uploadBlockStep(valid bool) {
	os.Setenv("PR_PROTECTOR_BLOCKED", strconv.FormatBool(valid))

	if valid {
		os.Exit(0)
	}

	log.Info("Uploading block step to pipeline")

	pipeline := getPipelineStr()
	// TODO maybe a better way to do this, like saving to temp file
	uploadCmd := fmt.Sprintf("echo '%s' | buildkite-agent pipeline upload", pipeline)
	log.Debugln("upload command: ", uploadCmd)
	out, err := exec.Command("/bin/bash", "-c", uploadCmd).CombinedOutput()

	if err != nil {
		log.Error("❌ Block step upload failed")
		log.Error(string(out))
		log.Error("Please open an issue on https://github.com/nickofthyme/pull-request-protector-buildkite-plugin/issues/new/choose")
		os.Exit(1)
	}

	log.Infoln("✅ Block step uploaded successfully")
}

func getPipelineStr() string {
	pipeline := SimplePipeline{
		Steps: []BlockStep{
			getPlugin().BlockStep,
		},
	}

	prettyJson(pipeline, "Block step pipeline config")

	pipelineBlob, err := json.Marshal(pipeline)

	if err != nil {
		log.Fatal(err)
	}

	return string(pipelineBlob)
}
