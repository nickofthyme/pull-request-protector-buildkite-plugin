package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/gobwas/glob"
	"github.com/google/go-github/v44/github"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

type GitHubAuth struct {
	AppID          int64
	InstallationID int64
	PrivateKey     []byte
}

func getGithubAuth() *GitHubAuth {
	authStr, err := getEnvString("GITHUB_AUTH", false)

	if err != nil {
		log.Fatal("GH_TOKEN or GITHUB_AUTH must be provided, found neither")
	}

	type GitHubAuthMapping struct {
		AppID          int64  `json:"appId"`
		InstallationID int64  `json:"installationId"`
		PrivateKey     string `json:"privateKey"`
	}
	var auth GitHubAuthMapping
	err = json.Unmarshal([]byte(authStr), &auth)

	if err != nil {
		log.Fatal(errors.Wrap(err, "Unable to parse GITHUB_AUTH json, please check json values."))
	}

	PrivateKey := []byte(fmt.Sprintf("%v", auth.PrivateKey))

	return &GitHubAuth{
		PrivateKey:     PrivateKey,
		AppID:          auth.AppID,
		InstallationID: auth.InstallationID,
	}
}

var githubClient = func() *github.Client {
	log.SetFlags(0)
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	tr := http.DefaultTransport

	token, err := getEnvString("GH_TOKEN", false)

	if err != nil {
		auth := getGithubAuth()

		itr, err := ghinstallation.New(
			tr,
			auth.AppID,
			auth.InstallationID,
			auth.PrivateKey,
		)

		// ctx := context.Background()
		// token, _ := itr.Token(ctx)

		// fmt.Println(token)

		if err != nil {
			log.Fatal(err)
		}

		return github.NewClient(&http.Client{Transport: itr})
	}

	if _, err := getEnvString("GITHUB_AUTH", false); err == nil {
		log.Println("Found both GH_TOKEN and GITHUB_AUTH, using GH_TOKEN")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc)
}()

func verifyMembership() bool {
	username := *jobEnvironment.Commit.Author.Login
	_, resp, err := githubClient.Organizations.GetOrgMembership(
		context.Background(),
		username,
		jobEnvironment.Owner,
	)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		log.Fatal(errors.Wrapf(err, "Provided credentials lack read access to %s organization", jobEnvironment.Owner))
	}

	if resp.StatusCode == http.StatusNotFound {
		log.Printf("❌ User %s is not a member of %s", username, jobEnvironment.Owner)
		return false
	}

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("✅ User %s is member of %s", username, jobEnvironment.Owner)
	return true
}

func verifyCollaborator() bool {
	username := *jobEnvironment.Commit.Author.Login
	_, resp, err := githubClient.Repositories.IsCollaborator(
		context.Background(),
		jobEnvironment.Owner,
		jobEnvironment.Repo,
		username,
	)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		log.Printf("❌ User %s is not a collaborator of %s", username, jobEnvironment.Repo)
		return false
	}

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("✅ User %s is collaborator of %s", username, jobEnvironment.Repo)
	return true
}

func verifyTeam(teams []string) bool {
	ctx := context.Background()
	username := *jobEnvironment.Commit.Author.Login

	for _, team := range teams {
		_, resp, err := githubClient.Teams.GetTeamBySlug(ctx, jobEnvironment.Owner, team)

		if resp.StatusCode == http.StatusNotFound {
			log.Printf("Could not find team %s in %s organization", team, jobEnvironment.Owner)
			return false
		}

		if err != nil {
			log.Fatal(err)
		}
	}

	for _, team := range teams {
		_, resp, err := githubClient.Teams.GetTeamMembershipBySlug(
			ctx,
			jobEnvironment.Owner,
			team,
			username,
		)
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			log.Printf("✅ User %s is member of team %s", username, team)
			return true
		}

		if err != nil && resp.StatusCode != http.StatusNotFound {
			log.Fatal(err)
		}
	}

	log.Printf("❌ User %s is not a member of any provided team", username)
	return false
}

func verifyCommitUser(userPatterns []string) bool {
	username := *jobEnvironment.Commit.Author.Login
	for _, pattern := range userPatterns {
		g := glob.MustCompile(pattern)

		if g.Match(username) {
			log.Printf("✅ User %s matches pattern %s", username, pattern)
			return true
		}
	}

	log.Printf("❌ User %s does not match any user pattern", username)
	return false
}

func verifyCommit() bool {
	verification := *jobEnvironment.Commit.Commit.Verification

	if verification.Verified == nil || !*verification.Verified {
		log.Printf("❌ Commit is not verified, reason: %s", *verification.Reason)
		return false
	}

	log.Println("✅ Commit is verified")
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

	files, resp, err := githubClient.PullRequests.ListFiles(
		ctx,
		jobEnvironment.Owner,
		jobEnvironment.Repo,
		int(jobEnvironment.PRNumber),
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

func prettyLog(v any) {
	s, _ := json.MarshalIndent(v, "", "\t")
	fmt.Print(string(s))
}
