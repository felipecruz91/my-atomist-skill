package main

import (
	"strings"
)

func ReplaceWithNewBaseImage(content, baseImage, newBaseImage string) string {
	return strings.ReplaceAll(content, "FROM "+baseImage, "FROM "+newBaseImage)
}
