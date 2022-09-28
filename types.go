package main

import (
	"github.com/atomist-skills/go-skill"
)

type OnDockerfile struct {
	Repository struct {
		Host string `edn:"docker.repository/host"`
		Name string `edn:"docker.repository/repository"`
	} `edn:"docker.file.from/repository"`
}

// OnCommit maps the incoming commit of the on_push and on_commit_signature to a Go struct
type OnCommit struct {
	Sha     string `edn:"git.commit/sha"`
	Message string `edn:"git.commit/message"`
	Repo    struct {
		Org struct {
			Name              string `edn:"git.org/name"`
			Url               string `edn:"git.provider/url"`
			GithubAccessToken string `edn:"github.org/installation-token"`
		} `edn:"git.repo/org"`
		SourceId string `edn:"git.repo/source-id"`
		Name     string `edn:"git.repo/name"`
	} `edn:"git.commit/repo"`
}

// GitRepoEntity provides mappings for a :git/repo entity
type GitRepoEntity struct {
	skill.Entity
	SourceId string `edn:"git.repo/source-id,omitempty"`
	Url      string `edn:"git.provider/url,omitempty"`
}

// GitCommitEntity provides mappings for a :git/commit entity
type GitCommitEntity struct {
	skill.Entity
	Sha  string        `edn:"git.commit/sha,omitempty"`
	Repo GitRepoEntity `edn:"git.commit/repo,omitempty"`
	Url  string        `edn:"git.provider/url,omitempty"`
}
