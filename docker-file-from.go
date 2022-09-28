package main

import (
	"context"
	"fmt"
	"github.com/google/go-github/v47/github"
	"golang.org/x/oauth2"
	"log"
	"strings"
	"time"

	"github.com/atomist-skills/go-skill"
	"github.com/atomist-skills/go-skill/util"
)

func handleDockerfileFrom(ctx context.Context, req skill.RequestContext) skill.Status {
	results := req.Event.Context.Subscription.Result

	commit := util.Decode[OnCommit](results[0][1])

	fmt.Printf("\nNew commit to repo %s in org %s\n", commit.Repo.Name, commit.Repo.Org)
	fmt.Printf("revision: %s\n", commit.Sha)
	fmt.Printf("message:  %s\n", commit.Message)

	baseAndNewImages := make(map[string]string)

	for _, result := range results {
		dockerfileFrom := util.Decode[OnDockerfile](result[0])
		fmt.Printf("dockerfileFrom:  %+v\n", dockerfileFrom)

		// If host is hub.docker.com, replace final image with Chainguard distroless image
		if dockerfileFrom.Repository.Host != "hub.docker.com" {
			continue
		}

		var newBaseImage string

		baseImage := dockerfileFrom.Repository.Name
		switch baseImage {
		case "alpine":
			newBaseImage = "cgr.dev/chainguard/alpine-base"
		case "busybox":
			newBaseImage = "cgr.dev/chainguard/busybox"
		case "golang":
			newBaseImage = "cgr.dev/chainguard/go"
		case "nginx":
			newBaseImage = "cgr.dev/chainguard/nginx"
		}

		fmt.Printf("newBaseImage:  %s\n", newBaseImage)

		// dockerfileFrom.DockerfileLineArgsString examples could be:
		// "nginx" or "alpine:3.11" or "golang:1.17-alpine as build"
		// so we have to get just the image name (first token)
		key := strings.Split(dockerfileFrom.DockerfileLineArgsString, " ")[0]
		baseAndNewImages[key] = newBaseImage // e.g. map["alpine:3.11"] = "cgr.dev/chainguard/alpine-base"
	}
	fmt.Println(baseAndNewImages)

	if len(baseAndNewImages) == 0 {
		return skill.Status{
			State:  skill.Info,
			Reason: fmt.Sprintf("unable to identify Chainguard distroless image replacement for %v", baseAndNewImages),
		}
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: commit.Repo.Org.GithubAccessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	sourceOwner := commit.Repo.Org.Name
	sourceRepo := commit.Repo.Name
	commitBranch := fmt.Sprintf("replace-docker-base-image-with-chainguard-distroless-%v", time.Now().UTC().Unix())
	baseBranch := "main"

	ref, err := getRef(ctx, client, sourceOwner, sourceRepo, commitBranch, baseBranch)
	if err != nil {
		log.Fatalf("Unable to get/create the commit reference: %s\n", err)
	}
	if ref == nil {
		log.Fatalf("No error where returned but the reference is nil")
	}

	sourceFiles := "Dockerfile"
	tree, err := getTree(ctx, client, ref, sourceFiles, sourceOwner, sourceRepo, commit.Repo.Org.GithubAccessToken, baseAndNewImages)
	if err != nil {
		log.Fatalf("Unable to create the tree based on the provided files: %s\n", err)
	}

	if err := pushCommit(ctx, client, ref, tree, sourceOwner, sourceRepo); err != nil {
		log.Fatalf("Unable to create the commit: %s\n", err)
	}

	if err := createPR(ctx, client,
		github.String("Replace Docker base image(s) with Chainguard distroless"),
		github.String(sourceOwner),
		github.String(sourceOwner),
		github.String(commitBranch),
		github.String(sourceRepo),
		github.String(sourceRepo),
		github.String(baseBranch),
		github.String(createPRBody(baseAndNewImages))); err != nil {
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
