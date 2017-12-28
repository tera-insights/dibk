package main

import (
	"dibk"
	"encoding/json"
	"fmt"
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
			Name:  "store",
			Usage: "Store a version of a binary",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "name"},
				cli.StringFlag{Name: "input"},
			},
			SkipFlagParsing: false,
			HideHelp:        false,
			Hidden:          false,
			Action: func(c *cli.Context) error {
				err := store(c)
				if err != nil {
					fmt.Printf("Error = %v\n", err)
					fmt.Println("Usage: dibk store --name OBJECT_NAME --input INPUT_FILE")
				}
				return err
			},
		},

		cli.Command{
			Name:  "retrieve",
			Usage: "Retrieve a version of a binary",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "name"},
				cli.IntFlag{Name: "version"},
				cli.StringFlag{Name: "output"},
			},
			SkipFlagParsing: false,
			HideHelp:        false,
			Hidden:          false,
			Action: func(c *cli.Context) error {
				err := retrieve(c)
				if err != nil {
					fmt.Printf("Error = %v\n", err)
					fmt.Println("Usage: dibk retrieve --name OBJECT_NAME --version OBJECT_VERSION --output OUTPUT_FILE")
				}
				return err
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
	name, inputPath, err := parseStoreFlags(c)
	if err != nil {
		return err
	}

	e, err := makeEngineFromConfig()
	if err != nil {
		return err
	}

	file, err := os.Open(inputPath)
	if err != nil {
		return err
	}

	return e.SaveObject(file, name)
}

func retrieve(c *cli.Context) error {
	name, version, outputPath, err := parseRetrieveFlags(c)
	if err != nil {
		return err
	}

	e, err := makeEngineFromConfig()
	if err != nil {
		return err
	}

	file, err := os.Open(outputPath)
	if err != nil {
		return err
	}

	return e.RetrieveObject(file, name, version)
}

func makeEngineFromConfig() (dibk.Engine, error) {
	conf, err := readConfig()
	if err != nil {
		return dibk.Engine{}, err
	}

	return dibk.MakeEngine(conf)
}

func parseStoreFlags(c *cli.Context) (string, string, error) {
	name := c.String("name")
	input := c.String("input")
	if name == "" {
		return name, input, fmt.Errorf("Could not find 'name' flag")
	}

	if input == "" {
		return name, input, fmt.Errorf("Could not find 'input' flag")
	}

	return name, input, nil
}

func parseRetrieveFlags(c *cli.Context) (name string, version int, output string, err error) {
	name = c.String("name")
	version = c.Int("version")
	output = c.String("output")

	if name == "" {
		err = fmt.Errorf("Could not find 'name' flag")
	}

	if version == 0 {
		err = fmt.Errorf("Could not find 'version'")
	}

	if output == "" {
		err = fmt.Errorf("Could not find 'output' flag")
	}

	return
}
