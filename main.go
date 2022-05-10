package main

import (
	"log"
	"os"
	"strconv"
)

func main() {
	valid := true
	plugin := getPlugin()

	log.Println("--- Verifing 3rd-party fork pull request")

	hasTargetFileChanges := checkTargetFileChanges(plugin.Files, nil)

	if !hasTargetFileChanges {
		log.Println("✅ No files matching any target pattern, skipping checks.")
		os.Exit(1)
	}

	log.Println("Found files matching one or more target pattern, validating checks.")

	if plugin.Verified {
		valid = verifyCommit()
	}

	if plugin.Users != nil {
		valid = verifyCommitUser(*plugin.Users)
	}

	if plugin.Teams != nil {
		valid = verifyTeam(*plugin.Teams)
	}

	if plugin.Member {
		valid = verifyMembership()
	}

	if plugin.Collaborator {
		valid = verifyCollaborator()
	}

	os.Setenv("PR_PROTECTOR_BLOCKED", strconv.FormatBool(valid))
}
