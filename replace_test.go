package main

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestReplaceWithNewBaseImage(t *testing.T) {
	input := `# syntax=docker/dockerfile:1.4
FROM golang:1.17-alpine as build

WORKDIR /work

COPY <<EOF go.mod
module hello
go 1.19
EOF

COPY <<EOF main.go
package main
import "fmt"
func main() {
    fmt.Println("Hello World!")
}
EOF
RUN go build -o hello .

FROM alpine:3.11

COPY --from=build /work/hello /hello
CMD ["/hello"]`

	expected := `# syntax=docker/dockerfile:1.4
FROM golang:1.17-alpine as build

WORKDIR /work

COPY <<EOF go.mod
module hello
go 1.19
EOF

COPY <<EOF main.go
package main
import "fmt"
func main() {
    fmt.Println("Hello World!")
}
EOF
RUN go build -o hello .

FROM cgr.dev/chainguard/alpine-base

COPY --from=build /work/hello /hello
CMD ["/hello"]`

	actual := ReplaceWithNewBaseImage(input, "alpine", "cgr.dev/chainguard/alpine-base")

	require.Equal(t, expected, actual)
}
