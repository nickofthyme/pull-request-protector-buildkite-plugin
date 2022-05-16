package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/gobwas/glob"
	"github.com/google/go-github/v44/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

type GitHubAuth struct {
	AppID          int64
	InstallationID int64
	PrivateKey     []byte
}

func getGithubAuth() *GitHubAuth {
	key := *getPlugin().GHAuthKey
	authStr, err := getEnvString(key, false)

	if err != nil {
		log.Fatalf("Unable to find environment variable for GitHub auth key as %s", key)
	}

	type GitHubAuthMapping struct {
		AppID          int64  `json:"appId"`
		InstallationID int64  `json:"installationId"`
		PrivateKey     string `json:"privateKey"`
	}

	var auth GitHubAuthMapping
	err = json.Unmarshal([]byte(authStr), &auth)

	if err != nil {
		log.Fatal(errors.Wrap(err, "Unable to parse GitHub auth json, please check json values"))
	}

	PrivateKey := []byte(fmt.Sprintf("%v", auth.PrivateKey))

	return &GitHubAuth{
		PrivateKey:     PrivateKey,
		AppID:          auth.AppID,
		InstallationID: auth.InstallationID,
	}
}

func getGithubClientS() *github.Client {
	tr := http.DefaultTransport

	if getPlugin().GHAuthKey != nil {
		auth := getGithubAuth()

		itr, err := ghinstallation.New(
			tr,
			auth.AppID,
			auth.InstallationID,
			auth.PrivateKey,
		)

		if err != nil {
			log.Fatal(err)
		}

		return github.NewClient(&http.Client{Transport: itr})
	}

	key := getPlugin().GHTokenKey
	token, err := getEnvString(key, false)

	if err != nil {
		log.Fatalf("Unable to find environment variable for GitHub token key as %s\n", key)
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc)
}

func verifyMembership() bool {
	username := *getJobEnv().Commit.Author.Login
	_, resp, err := getGithubClient().Organizations.GetOrgMembership(
		context.Background(),
		username,
		getJobEnv().Owner,
	)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		log.Fatal(errors.Wrapf(err, "Provided credentials lack read access to %s organization", getJobEnv().Owner))
	}

	if resp.StatusCode == http.StatusNotFound {
		log.Errorf("❌ User %s is not a member of %s organization", username, getJobEnv().Owner)
		return false
	}

	if err != nil {
		log.Fatal(err)
	}

	log.Infof("✅ User %s is a member of %s organization", username, getJobEnv().Owner)
	return true
}

func verifyCollaborator() bool {
	username := *getJobEnv().Commit.Author.Login
	_, resp, err := getGithubClient().Repositories.IsCollaborator(
		context.Background(),
		getJobEnv().Owner,
		getJobEnv().Repo,
		username,
	)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		log.Errorf("❌ User %s is not a collaborator of the %s repo", username, getJobEnv().Repo)
		return false
	}

	if err != nil {
		log.Fatal(err)
	}

	log.Infof("✅ User %s is a collaborator of the %s repo", username, getJobEnv().Repo)
	return true
}

func getVerifiedTeams() []string {
	ctx := context.Background()
	verifiedTeams := []string{}
	for _, team := range *getPlugin().Teams {
		_, resp, err := getGithubClient().Teams.GetTeamBySlug(ctx, getJobEnv().Owner, team)

		if resp.StatusCode == http.StatusOK {
			verifiedTeams = append(verifiedTeams, team)
		} else if resp.StatusCode == http.StatusNotFound {
			log.Warnf("Could not find team %s in %s organization, ignoring team", team, getJobEnv().Owner)
		} else if err != nil {
			log.Warnf("Error looking up team %s in %s organization, ignoring team", team, getJobEnv().Owner)
			log.Warn(err)
		}
	}

	return verifiedTeams
}

func verifyTeamMembership() bool {
	teams := getVerifiedTeams()
	ctx := context.Background()
	username := *getJobEnv().Commit.Author.Login

	for _, team := range teams {
		_, resp, err := getGithubClient().Teams.GetTeamMembershipBySlug(
			ctx,
			getJobEnv().Owner,
			team,
			username,
		)
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			log.Infof("✅ User %s is a member of the %s team", username, team)
			return true
		}

		if err != nil && resp.StatusCode != http.StatusNotFound {
			log.Fatal(err)
		}
	}

	log.Errorf("❌ User %s is not a member of any provided team", username)
	return false
}

func verifyCommitUser() bool {
	username := *getJobEnv().Commit.Author.Login
	for _, uname := range *getPlugin().Users {
		if username == uname {
			log.Infof("✅ User %s is an allowed user", username)
			return true
		}
	}

	log.Errorf("❌ User %s is not an allowed user", username)
	return false
}

func verifyCommit() bool {
	verification := *getJobEnv().Commit.Commit.Verification

	if verification.Verified == nil || !*verification.Verified {
		log.Errorf("❌ Commit is not verified (reason: %s)", *verification.Reason)
		return false
	}

	log.Infoln("✅ Commit is verified")
	return true
}

func checkTargetFileChanges(filePatterns []string, opts *github.ListOptions) bool {
	ctx := context.Background()
	if opts == nil {
		opts = &github.ListOptions{
			PerPage: 30,
			Page:    1,
		}
	}

	files, resp, err := getGithubClient().PullRequests.ListFiles(
		ctx,
		getJobEnv().Owner,
		getJobEnv().Repo,
		int(getJobEnv().PRNumber),
		opts,
	)
	defer resp.Body.Close()

	if err != nil {
		log.Fatal(errors.Wrap(err, "Failed to get files for pull request.\n"))
	}

	for _, pattern := range filePatterns {
		g := glob.MustCompile(pattern, '/')

		for _, file := range files {
			match := g.Match(*file.Filename)

			if match {
				return true
			}
		}
	}

	if resp.NextPage == 0 {
		return false
	}

	opts.Page = resp.NextPage

	return checkTargetFileChanges(filePatterns, opts)
}

func prettyJson(v any, label string) {
	s, _ := json.MarshalIndent(v, "", "    ")
	log.Debugf("%s:\n%s", label, string(s))
}

var githubClient *github.Client

func getGithubClient() *github.Client {
	if githubClient == nil {
		githubClient = getGithubClientS()
	}
	return githubClient
}
