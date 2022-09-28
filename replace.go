package main

import (
	"fmt"
	"regexp"
)

func ReplaceWithNewBaseImage(content, baseImage, newBaseImage string) string {
	var re = regexp.MustCompile(fmt.Sprintf(`(?m)FROM %s?:(.*)`, baseImage))
	return re.ReplaceAllString(content, "FROM "+newBaseImage)
}
