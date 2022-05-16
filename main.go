package main

import (
	"os"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

func initPlugin() {
	setupLogger()
	log.Infoln("Verifing 3rd-party fork pull request")

	prettyJson(getPlugin(), "Plugin config")

	// TODO make this better for release
	if os.Getenv("CI") != "true" {
		err := godotenv.Load()
		if err != nil {
			log.Fatal(err, "Error loading .env file")
		}
	}
}

func setupLogger() {
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp:       true,
		DisableLevelTruncation: true,
		ForceQuote:             true,
	})

	debugKey := "BUILDKITE_PLUGIN_PULL_REQUEST_PROTECTOR_DEBUG"
	if os.Getenv(debugKey) == "true" || os.Getenv("CI") != "true" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

func main() {
	initPlugin()

	valid := true
	shouldCheck := isCheckRequired()

	if !shouldCheck {
		commitValid := getPlugin().Verified && verifyCommit()
		uploadBlockStep(commitValid)
	}

	hasTargetFileChanges := checkTargetFileChanges(getPlugin().Files, nil)

	if !hasTargetFileChanges {
		log.Infoln("✅ No files matching any target pattern, skipping checks")
		os.Exit(1)
	}

	log.Infoln("Found files matching one or more target pattern, validating checks")

	if valid && getPlugin().Verified {
		valid = verifyCommit()
	}

	if valid && getPlugin().Users != nil {
		valid = verifyCommitUser()
	}

	if valid && getPlugin().Teams != nil {
		valid = verifyTeamMembership()
	}

	if valid && getPlugin().Member {
		valid = verifyMembership()
	}

	if valid && getPlugin().Collaborator {
		valid = verifyCollaborator()
	}

	uploadBlockStep(valid)
}
