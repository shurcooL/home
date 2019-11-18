// +build ignore

package main

import (
	"log"

	"github.com/shurcooL/vfsgen"
)

import "github.com/shurcooL/home/internal/exp/app/notifications/assets"

func main() {
	err := vfsgen.Generate(assets.Assets, vfsgen.Options{
		PackageName:     "assets",
		BuildTags:       "!dev",
		VariableName:    "Assets",
		VariableComment: "Assets contains assets for notifications app.",
	})
	if err != nil {
		log.Fatalln(err)
	}
}
