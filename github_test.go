package main

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_createPRBody(t *testing.T) {
	baseAndNewImages := map[string]string{
		"alpine:3.11":        "cgr.dev/chainguard/alpine-base",
		"golang:1.17-alpine": "cgr.dev/chainguard/go",
	}

	expected := `This pull request replaces the following base image(s):
- the Docker base image ` + "`alpine:3.11`" + ` to ` + "`cgr.dev/chainguard/alpine-base`" + `
- the Docker base image ` + "`golang:1.17-alpine`" + ` to ` + "`cgr.dev/chainguard/go`" + `

---

Chainguard Images is a collection of container images designed for **minimalism** and **security**.

Many of these images are **distroless**; they contain only an application and its runtime dependencies. There is no shell or package manager.

They provide **SBOM support** and **signatures** for known provenance and more secure base images.
`

	actual := createPRBody(baseAndNewImages)

	require.Equal(t, expected, actual)
}
