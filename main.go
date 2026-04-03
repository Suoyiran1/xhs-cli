package main

import (
	"os"

	"github.com/Suoyiran1/xhs-cli/cmd"
	"github.com/Suoyiran1/xhs-cli/internal/configs"
)

func main() {
	if binPath := os.Getenv("ROD_BROWSER_BIN"); binPath != "" {
		configs.SetBinPath(binPath)
	}
	cmd.Execute()
}
