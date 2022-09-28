package main

import (
	"context"
	"fmt"
	"github.com/google/go-github/v47/github"
	"golang.org/x/oauth2"
	"log"

	"github.com/atomist-skills/go-skill"
	"github.com/atomist-skills/go-skill/util"
)

func handleDockerfileFrom(ctx context.Context, req skill.RequestContext) skill.Status {
	result := req.Event.Context.Subscription.Result[0]
	dockerfileFrom := util.Decode[OnDockerfile](result[0])
	commit := util.Decode[OnCommit](result[1])

	fmt.Printf("\nNew commit to repo %s in org %s\n", commit.Repo.Name, commit.Repo.Org)
	fmt.Printf("revision: %s\n", commit.Sha)
	fmt.Printf("message:  %s\n", commit.Message)
	fmt.Printf("author name:  %s\n", commit.Author.Name)
	fmt.Printf("author login:  %s\n", commit.Author.Login)
	fmt.Printf("author login:  %s\n", commit.Author.Login)
	fmt.Printf("dockerfileFrom:  %+v\n", dockerfileFrom)
	fmt.Printf("commit.Repo.Org.GithubAccessToken:  %s\n", commit.Repo.Org.GithubAccessToken)
	// TODO: If host is hub.docker.com, replace final image with Chainguard distroless image
	if dockerfileFrom.Repository.Host != "hub.docker.com" {
		return skill.Status{
			State:  skill.Completed,
			Reason: "No base image from hub.docker.com",
		}
	}

	baseImage := dockerfileFrom.Repository.Name
	var newBaseImage string

	switch baseImage {
	case "alpine":
		newBaseImage = "cgr.dev/chainguard/alpine-base"
	}

	fmt.Printf("newBaseImage:  %s\n", newBaseImage)

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: commit.Repo.Org.GithubAccessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	sourceOwner := "felipecruz91" // TODO: get GH repo owner from .edn
	sourceRepo := commit.Repo.Name
	commitBranch := "replace-docker-base-image-with-chainguard-distroless"
	baseBranch := "main"

	ref, err := getRef(ctx, client, sourceOwner, sourceRepo, commitBranch, baseBranch)
	if err != nil {
		log.Fatalf("Unable to get/create the commit reference: %s\n", err)
	}
	if ref == nil {
		log.Fatalf("No error where returned but the reference is nil")
	}

	sourceFiles := "Dockerfile"
	tree, err := getTree(ctx, client, ref, sourceFiles, sourceOwner, sourceRepo, commit.Repo.Org.GithubAccessToken, baseImage, newBaseImage)
	if err != nil {
		log.Fatalf("Unable to create the tree based on the provided files: %s\n", err)
	}

	if err := pushCommit(ctx, client, ref, tree, sourceOwner, sourceRepo); err != nil {
		log.Fatalf("Unable to create the commit: %s\n", err)
	}

	if err := createPR(ctx, client,
		github.String(fmt.Sprintf("Replace Docker base image from %s to Chainguard distroless", dockerfileFrom.Repository.Name)),
		github.String(sourceOwner),
		github.String(sourceOwner),
		github.String(commitBranch),
		github.String(sourceRepo),
		github.String(sourceRepo),
		github.String(baseBranch),
		github.String("Body message")); err != nil {
		log.Fatalf("Error while creating the pull request: %s", err)
	}
	if err != nil {
		return skill.Status{
			State:  skill.Error,
			Reason: err.Error(),
		}
	}

	return skill.Status{
		State:  skill.Completed,
		Reason: "Handled Git push",
	}
}
