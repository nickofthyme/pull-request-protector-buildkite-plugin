package main

import (
	"encoding/json"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/google/go-github/v44/github"
	"github.com/pkg/errors"
	"github.com/xeipuuv/gojsonschema"
)

const pluginName = "github.com/nickofthyme/pull-request-protector-buildkite-plugin"

type Plugin struct {
	Users        *[]string `json:"users,omitempty"`
	Teams        *[]string `json:"teams,omitempty"`
	Member       bool      `json:"member"`
	Collaborator bool      `json:"collaborator"`
	Verified     bool      `json:"verified"`
	Files        []string  `json:"files"`
	BlockStep    BlockStep `json:"block_step"`
	GHTokenKey   string    `json:"gh_token_env"`
	Debug        bool      `json:"debug"`
	GHAuthKey    *string   `json:"gh_auth_env"`
}

var defaultBlockStep = BlockStep{
	Prompt: github.String("Default block step"),
	Type:   "block",
}

func getPluginS() *Plugin {
	pluginMappings := []map[string]interface{}{}

	pluginStr, _ := getEnvString("BUILDKITE_PLUGINS", true)
	json.Unmarshal([]byte(pluginStr), &pluginMappings)

	prpPlugin := &Plugin{
		Debug:        false,
		Member:       true,
		Collaborator: true,
		Verified:     true,
		GHTokenKey:   "GITHUB_TOKEN",
		Files:        []string{".buildkite/**"},
		BlockStep:    defaultBlockStep,
	}

	for _, plugins := range pluginMappings {
		for key, plugin := range plugins {
			if strings.HasPrefix(key, pluginName) {
				pluginBlob, errMarshal := json.Marshal(plugin)

				if errMarshal != nil {
					log.Fatalln(errors.Wrap(errMarshal, "failed to marshal pull-request-protector plugin configuration"))
				}

				err := json.Unmarshal(pluginBlob, &prpPlugin)

				if err != nil {
					log.Fatalln(errors.Wrap(err, "failed to parse pull-request-protector plugin configuration"))
				}

				validateBlockStep(prpPlugin.BlockStep)

				break
			}
		}
	}

	if prpPlugin == nil {
		log.Fatalln("Error parsing pull-request-protector plugin configuration")
	}

	return prpPlugin
}

func validateBlockStep(blockConfig BlockStep) {
	stepJsonBlob, err := json.Marshal(blockConfig)

	prettyJson(blockConfig, "Block step config")

	if err != nil {
		log.Fatalln(errors.Wrap(err, "Error attemptting to Marshal block step blob"))
	}

	stepStr := string(stepJsonBlob)
	schemaLoader := gojsonschema.NewReferenceLoader("https://raw.githubusercontent.com/buildkite/pipeline-schema/master/schema.json#/definitions/blockStep")
	documentLoader := gojsonschema.NewStringLoader(stepStr)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)

	if err != nil {
		log.Fatalln(errors.Wrap(err, "Error attemptting to validate block step from schema"))
	}

	if !result.Valid() {
		log.Error("❌ The block_step provided fails schema validation, see errors below:\n")
		for _, desc := range result.Errors() {
			log.Errorf("   - %s\n", desc)
		}
		os.Exit(1)
	}
}

var plugin *Plugin

func getPlugin() *Plugin {
	if plugin == nil {
		plugin = getPluginS()
	}
	return plugin
}
