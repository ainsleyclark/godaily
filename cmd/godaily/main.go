package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/ainsleyclark/godaily/internal/news"
	_ "github.com/ainsleyclark/godaily/internal/source"
	"github.com/urfave/cli/v3"
)

var cmd = &cli.Command{
	Name:  "godaily",
	Usage: "Daily Go news, straight to your inbox",
	Commands: []*cli.Command{
		{
			Name:  "run",
			Usage: "Gather all news from sources.",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return nil
			},
		},
		{
			Name:  "sources",
			Usage: "Lists registered source names",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				for _, name := range news.Sources {
					fmt.Println(name)
				}
				return nil
			},
		},
		{
			Name: "fetch",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "provider",
					Usage: "Provider of source information",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				fetcher, err := news.Get(news.Source(cmd.String("provider")))
				if err != nil {
					return err
				}

				items, err := fetcher.Fetch(ctx)
				if err != nil {
					return err
				}

				indent, err := json.MarshalIndent(items, "", "  ")
				if err != nil {
					return err
				}

				fmt.Println(string(indent))
				return nil
			},
		},
	},
}

func main() {
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
