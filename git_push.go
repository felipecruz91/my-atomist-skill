package main

import (
	"context"
	"fmt"

	"github.com/atomist-skills/go-skill"
	"github.com/atomist-skills/go-skill/util"
)

func handleGitPush(ctx context.Context, req skill.RequestContext) skill.Status {
	result := req.Event.Context.Subscription.Result[0]
	commit := util.Decode[OnCommit](result[0])

	fmt.Printf("\nNew commit to repo %s\n", commit.Repo.Name)
	fmt.Printf("revision: %s\n", commit.Sha)
	fmt.Printf("message:  %s\n", commit.Message)
	fmt.Printf("author name:  %s\n", commit.Author.Name)
	fmt.Printf("author login:  %s\n", commit.Author.Login)

	return skill.Status{
		State:  skill.Completed,
		Reason: "Handled Git push",
	}
}
