package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/go-github/v47/github"
	"net/http"
	"strings"
	"time"
)

type RepositoryFileContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Sha         string `json:"sha"`
	Size        int    `json:"size"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	GitURL      string `json:"git_url"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	Encoding    string `json:"encoding"`
	Links       struct {
		Self string `json:"self"`
		Git  string `json:"git"`
		HTML string `json:"html"`
	} `json:"_links"`
}

// getRef returns the commit branch reference object if it exists or creates it
// from the base branch before returning it.
func getRef(ctx context.Context, client *github.Client, sourceOwner, sourceRepo, commitBranch, baseBranch string) (ref *github.Reference, err error) {
	if ref, _, err = client.Git.GetRef(ctx, sourceOwner, sourceRepo, "refs/heads/"+commitBranch); err == nil {
		return ref, nil
	}

	// We consider that an error means the branch has not been found and needs to
	// be created.
	if commitBranch == baseBranch {
		return nil, errors.New("the commit branch does not exist but `-base-branch` is the same as `-commit-branch`")
	}

	if baseBranch == "" {
		return nil, errors.New("the `-base-branch` should not be set to an empty string when the branch specified by `-commit-branch` does not exists")
	}

	var baseRef *github.Reference
	if baseRef, _, err = client.Git.GetRef(ctx, sourceOwner, sourceRepo, "refs/heads/"+baseBranch); err != nil {
		return nil, err
	}
	newRef := &github.Reference{Ref: github.String("refs/heads/" + commitBranch), Object: &github.GitObject{SHA: baseRef.Object.SHA}}
	ref, _, err = client.Git.CreateRef(ctx, sourceOwner, sourceRepo, newRef)
	return ref, err
}

// getTree generates the tree to commit based on the given files and the commit
// of the ref you got in getRef.
func getTree(ctx context.Context, client *github.Client, ref *github.Reference, sourceFiles, sourceOwner, sourceRepo, accessToken string, baseAndNewImages map[string]string) (tree *github.Tree, err error) {
	// Create a tree with what to commit.
	entries := []*github.TreeEntry{}

	// Load each file into the tree.
	for _, fileArg := range strings.Split(sourceFiles, ",") {
		file, content, err := getFileContent(fileArg, sourceOwner, sourceRepo, accessToken)
		if err != nil {
			return nil, err
		}
		replacedContent := string(content)
		for baseImage, newBaseImage := range baseAndNewImages {
			replacedContent = ReplaceWithNewBaseImage(replacedContent, baseImage, newBaseImage)
		}

		entries = append(entries, &github.TreeEntry{Path: github.String(file), Type: github.String("blob"), Content: github.String(replacedContent), Mode: github.String("100644")})
	}

	tree, _, err = client.Git.CreateTree(ctx, sourceOwner, sourceRepo, *ref.Object.SHA, entries)
	return tree, err
}

// getFileContent loads the local content of a file and return the target name
// of the file in the target repository and its contents.
func getFileContent(fileArg, sourceOwner, sourceRepo, accessToken string) (targetName string, b []byte, err error) {
	var localFile string
	files := strings.Split(fileArg, ":")
	switch {
	case len(files) < 1:
		return "", nil, errors.New("empty `-files` parameter")
	case len(files) == 1:
		localFile = files[0]
		targetName = files[0]
	default:
		localFile = files[0]
		targetName = files[1]
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", sourceOwner, sourceRepo, localFile), nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	repoFileContent := RepositoryFileContent{}
	if err := json.NewDecoder(resp.Body).Decode(&repoFileContent); err != nil {
		return "", nil, err
	}

	rawDecodedFileContent, err := base64.StdEncoding.DecodeString(repoFileContent.Content)
	if err != nil {
		return "", nil, err
	}

	return targetName, rawDecodedFileContent, err
}

// pushCommit creates the commit in the given reference using the given tree.
func pushCommit(ctx context.Context, client *github.Client, ref *github.Reference, tree *github.Tree, sourceOwner, sourceRepo string) (err error) {
	// Get the parent commit to attach the commit to.
	parent, _, err := client.Repositories.GetCommit(ctx, sourceOwner, sourceRepo, *ref.Object.SHA, nil)
	if err != nil {
		return err
	}
	// This is not always populated, but is needed.
	parent.Commit.SHA = parent.SHA

	// Create the commit using the tree.
	authorName := "atomist-bot"
	authorEmail := "bot@atomist.com"
	commitMessage := "Replace Docker base image(s)"
	date := time.Now()
	author := &github.CommitAuthor{Date: &date, Name: &authorName, Email: &authorEmail}
	commit := &github.Commit{Author: author, Message: &commitMessage, Tree: tree, Parents: []*github.Commit{parent.Commit}}
	newCommit, _, err := client.Git.CreateCommit(ctx, sourceOwner, sourceRepo, commit)
	if err != nil {
		return err
	}

	// Attach the commit to the master branch.
	ref.Object.SHA = newCommit.SHA
	_, _, err = client.Git.UpdateRef(ctx, sourceOwner, sourceRepo, ref, false)
	return err
}

// createPR creates a pull request. Based on: https://godoc.org/github.com/google/go-github/github#example-PullRequestsService-Create
func createPR(ctx context.Context, client *github.Client, prSubject, prRepoOwner, sourceOwner, commitBranch, prRepo, sourceRepo, prBranch, prDescription *string) (err error) {
	if *prSubject == "" {
		return errors.New("missing `-pr-title` flag; skipping PR creation")
	}

	if *prRepoOwner != "" && *prRepoOwner != *sourceOwner {
		*commitBranch = fmt.Sprintf("%s:%s", *sourceOwner, *commitBranch)
	} else {
		prRepoOwner = sourceOwner
	}

	if *prRepo == "" {
		prRepo = sourceRepo
	}

	newPR := &github.NewPullRequest{
		Title:               prSubject,
		Head:                commitBranch,
		Base:                prBranch,
		Body:                prDescription,
		MaintainerCanModify: github.Bool(true),
	}

	pr, _, err := client.PullRequests.Create(ctx, *prRepoOwner, *prRepo, newPR)
	if err != nil {
		return err
	}

	fmt.Printf("PR created: %s\n", pr.GetHTMLURL())
	return nil
}

func createPRBody(baseAndNewImages map[string]string) string {
	var sb strings.Builder
	sb.WriteString("This pull request replaces the following base image(s):\n")

	for baseImage, newBaseImage := range baseAndNewImages {
		sb.WriteString(fmt.Sprintf("- the Docker base image `%s` to `%s`", baseImage, newBaseImage))
		sb.WriteString("\n")
	}

	sb.WriteString(`
---

Chainguard Images is a collection of container images designed for **minimalism** and **security**.

Many of these images are **distroless**; they contain only an application and its runtime dependencies. There is no shell or package manager.

They provide **SBOM support** and **signatures** for known provenance and more secure base images.
`)

	return sb.String()
}
