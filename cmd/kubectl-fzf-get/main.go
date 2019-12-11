package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/at-ishikawa/kubectl-fzf-get/internal/command"
)

func main() {
	var outputFormat string
	previewFormat := flag.String("preview-format", "describe", "The format of preview")
	flag.StringVar(&outputFormat, "output", "name", "The output format")
	flag.StringVar(&outputFormat, "o", "name", "The output format")
	helpFlag := flag.Bool("help", false, "This help message")
	flag.Parse()
	if *helpFlag {
		flag.PrintDefaults()
		return
	}

	c, err := command.NewCli(flag.Args(), *previewFormat, outputFormat)
	if err != nil {
		fmt.Println(err)
		flag.PrintDefaults()
		os.Exit(1)
	}

	exitCode, err := c.Run(context.Background())
	if err != nil {
		fmt.Println(err)
	}
	os.Exit(exitCode)
}
