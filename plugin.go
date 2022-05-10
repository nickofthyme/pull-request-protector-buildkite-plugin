package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/pkg/errors"
	"github.com/xeipuuv/gojsonschema"
)

const pluginName = "github.com/nickofthyme/pull-request-protector-buildkite-plugin"

type Plugin struct {
	Users        *[]string    `json:"users,omitempty"`
	Teams        *[]string    `json:"teams,omitempty"`
	Member       bool         `json:"member,omitempty"`
	Collaborator bool         `json:"collaborator,omitempty"`
	Verified     bool         `json:"verified,omitempty"`
	Files        []string     `json:"files,omitempty"`
	BlockStep    *interface{} `json:"block_step,omitempty"`
}

func getPlugin() Plugin {
	pluginMappings := []map[string]interface{}{}

	pluginStr, _ := getEnvString("BUILDKITE_PLUGINS", true)
	json.Unmarshal([]byte(pluginStr), &pluginMappings)

	prpPlugin := &Plugin{
		Member:       true,
		Collaborator: true,
		Verified:     true,
		Files:        []string{".buildkite/**"},
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

	return *prpPlugin
}

func validateBlockStep(blockConfig *interface{}) {
	if blockConfig == nil {
		return
	}
	stepJsonBlob, err := json.Marshal(blockConfig)

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
		// TODO improve error output
		fmt.Printf("The document is not valid. see errors :\n")
		for _, desc := range result.Errors() {
			fmt.Println(desc.Field())
			fmt.Printf("- %s\n", desc)
		}
	}
}
