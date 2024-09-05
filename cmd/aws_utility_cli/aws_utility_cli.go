package main

import (
	"aws_utility/pkg/clicommands"
	"aws_utility/pkg/logger"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/urfave/cli/v2"
)

func main() {
	logger.Init()

	app := &cli.App{
		Name:  "aws_utility_cli",
		Usage: "AWS Utility CLI",
		Commands: []*cli.Command{
			{
				Name:    "lambda",
				Aliases: []string{"l"},
				Usage:   "Execute a Lambda function",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "cluster", Usage: "Cluster name"},
					&cli.StringFlag{Name: "service", Usage: "Service name"},
					&cli.StringFlag{Name: "tag", Usage: "Tag name"},
				},
				Action: func(c *cli.Context) error {
					profile := c.String("profile")
					lambdaName := c.Args().First()
					cluster := c.String("cluster")
					service := c.String("service")
					tag := c.String("tag")
					return clicommands.ExecuteLambda(profile, lambdaName, cluster, service, tag)
				},
			},
			{
				Name:  "list_lambdas",
				Usage: "List available Lambda functions",
				Action: func(c *cli.Context) error {
					profile := c.String("profile")
					return clicommands.ListLambdas(profile)
				},
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "profile",
				Aliases: []string{"p"},
				Usage:   "AWS SSO profile name",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return runCharmInterface()
			}
			return cli.ShowAppHelp(c)
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		logger.Error(err)
		os.Exit(1)
	}
}

func runCharmInterface() error {
	initialModel := clicommands.InitialModel()
	p := tea.NewProgram(initialModel)
	_, err := p.Run()
	return err
}
