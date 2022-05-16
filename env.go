package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/google/go-github/v44/github"
	"github.com/pkg/errors"
)

type JobEnvironment struct {
	Owner      string
	Repo       string
	PRUsername string
	PRNumber   int64
	Commit     *github.RepositoryCommit
	Files      []*github.CommitFile
}

func getEnvString(key string, required bool) (string, error) {
	val, ok := os.LookupEnv(key)

	if !ok || val == "" {
		if required {
			if !ok {
				log.Fatalf("%s is required but was not found\n", key)
			}
			log.Fatalf("%s is required and found to be empty\n", key)
		}

		if !ok {
			return "", errors.New(fmt.Sprintf("%s not found\n", key))
		}

		return "", errors.New(fmt.Sprintf("%s found to be empty\n", key))
	}

	return val, nil
}

func getEnvInt(key string, required bool) (int64, error) {
	strValue, strErr := getEnvString(key, required)

	if strErr != nil {
		return 0, strErr
	}

	val, err := strconv.ParseInt(strValue, 10, 64)

	if err != nil && required {
		log.Fatal(errors.Wrap(err, fmt.Sprintf("%s is required but failed to cast as int\n", key)))
	}

	return val, err
}

// Returns value at array index or errors
func getArrayValue[V any](arr []V, index int, errMsg string) V {
	if len(arr) <= index {
		log.Fatalln(errMsg)
	}

	return arr[index]
}

func isCheckRequired() bool {
	isPr, _ := getEnvString("BUILDKITE_PULL_REQUEST", false)

	if isPr == "false" {
		// TODO handle this flow
		log.Infoln("Build is not a pull request")
		return false
	}

	re := regexp.MustCompile("^(?:git|https)://(.+?)$")
	baseRepo := re.ReplaceAllString(os.Getenv("BUILDKITE_REPO"), "$1")
	prRepo := re.ReplaceAllString(os.Getenv("BUILDKITE_PULL_REQUEST_REPO"), "$1")

	if baseRepo == prRepo {
		log.Infoln("Pull request is not from a fork")
		return false
	}

	return true
}

func getJobEnvironment() *JobEnvironment {
	repoUrl, _ := getEnvString("BUILDKITE_REPO", true)
	commitSha, _ := getEnvString("BUILDKITE_COMMIT", true)
	prNumber, _ := getEnvInt("BUILDKITE_PULL_REQUEST", true)

	re := regexp.MustCompile(`^(?:git@github\.com:|https:\/\/github.com\/)([^#/.]+)\/([^#/.]+)\.git`)
	subMatches := re.FindStringSubmatch(repoUrl)

	owner := getArrayValue(subMatches, 1, "Could not determine owner from BUILDKITE_REPO")
	repo := getArrayValue(subMatches, 2, "Could not determine repo from BUILDKITE_REPO")

	ctx := context.Background()

	commit, commitResp, commitErr := getGithubClient().Repositories.GetCommit(ctx, owner, repo, commitSha, nil)
	defer commitResp.Body.Close()

	if commitErr != nil {
		log.Fatal(errors.Wrap(commitErr, "Failed to get job commit"))
	}

	files, filesResp, filesErr := getGithubClient().PullRequests.ListFiles(ctx, owner, repo, int(prNumber), nil)
	defer filesResp.Body.Close()

	if filesErr != nil {
		log.Fatal("Failed to get files for pull request")
	}

	return &JobEnvironment{
		Owner:    owner,
		Repo:     repo,
		Commit:   commit,
		Files:    files,
		PRNumber: prNumber,
	}
}

var jobEnvironment *JobEnvironment

func getJobEnv() *JobEnvironment {
	if jobEnvironment == nil {
		jobEnvironment = getJobEnvironment()
	}
	return jobEnvironment
}
