package main

import (
	"os"

	"github.com/urfave/cli/v2"
	"github.com/w-h-a/backend/cmd"
)

func main() {
	app := &cli.App{
		Name: "backend",
		Commands: []*cli.Command{
			{
				Name: "backend",
				Action: func(ctx *cli.Context) error {
					return cmd.Run(ctx)
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		// log
		os.Exit(1)
	}
}
