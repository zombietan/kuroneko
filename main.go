package main

import (
	"os"

	"github.com/fatih/color"
	"github.com/zombietan/kuroneko/cmd"
)

func main() {
	cmd.Execute(color.Output, color.Error, os.Args[1:]).Exit()
}
