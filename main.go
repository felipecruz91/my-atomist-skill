package main

import "github.com/atomist-skills/go-skill"

func main() {
	skill.Start(skill.Handlers{
		"docker-file-with-non-distroless-from": handleDockerfileFrom,
	})
}
