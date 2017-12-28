package main

import (
	"dibk"
	"encoding/json"
	"os"
	"time"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Description = "Disk image backup"
	app.Name = "dibk"
	app.Version = "0.1.0"
	app.Compiled = time.Now()
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "Jess Smith",
			Email: "smith.jessk@gmail.com",
		},
	}
	app.Commands = []cli.Command{
		cli.Command{
			Name:      "store",
			Usage:     "Store a version of a binary",
			UsageText: "store - Store a version of a binary",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "name"},
				cli.StringFlag{Name: "input"},
			},
			SkipFlagParsing: false,
			HideHelp:        false,
			Hidden:          false,
			Action: func(c *cli.Context) error {
				return store(c)
			},
		},
	}

	app.Action = func(c *cli.Context) error {
		cli.ErrWriter.Write([]byte("Invalid subcommand. Options are:\n"))
		cli.DefaultAppComplete(c)
		return nil
	}

	app.Run(os.Args)
}

func readConfig() (dibk.Configuration, error) {
	file, err := os.Open("dibk_config.json")
	if err != nil {
		return dibk.Configuration{}, err
	}

	decoder := json.NewDecoder(file)
	conf := dibk.Configuration{}
	err = decoder.Decode(&conf)
	return conf, err
}

func store(c *cli.Context) error {
	conf, err := readConfig()
	if err != nil {
		return err
	}

	_, err = dibk.MakeEngine(conf)
	if err != nil {
		return err
	}
	return nil
}
