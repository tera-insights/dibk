package main

import (
	"dibk"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/urfave/cli"
)

func main() {
	app := buildApp()
	app.Run(os.Args)
}

func buildApp() *cli.App {
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
		buildStoreCommand(),
		buildRetrieveCommand(),
	}

	app.Action = func(c *cli.Context) error {
		cli.ErrWriter.Write([]byte("Invalid subcommand. Options are:\n"))
		cli.DefaultAppComplete(c)
		return nil
	}

	return app
}

func store(c *cli.Context) error {
	name, inputPath, err := parseStoreFlags(c)
	if err != nil {
		return err
	}

	e, err := makeEngineFromContext(c)
	if err != nil {
		return err
	}

	file, err := e.OpenFileForReading(inputPath)
	if err != nil {
		return err
	}

	return e.SaveObject(file, name, c.Int("mbperblock")*1024*1024)
}

func retrieve(c *cli.Context) error {
	name, version, outputPath, err := parseRetrieveFlags(c)
	if err != nil {
		return err
	}

	e, err := makeEngineFromContext(c)
	if err != nil {
		return err
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}

	return e.RetrieveObject(file, name, version)
}

func makeEngineFromContext(c *cli.Context) (dibk.Engine, error) {
	return dibk.MakeEngine(dibk.Configuration{
		DBPath:            c.String("db"),
		StorageLocation:   c.String("storage"),
		IsDirectIOEnabled: c.Bool("directio"),
	})
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

func getCommonSubcommandFlags() []cli.Flag {
	return []cli.Flag{
		cli.BoolFlag{Name: "directio"},
		cli.StringFlag{Name: "db"},
		cli.StringFlag{Name: "storage"},
	}
}

func buildRequiredFlagText(flags []string) string {
	s := ""
	for _, f := range flags {
		s += fmt.Sprintf("--%s $%s ", f, strings.ToUpper(f))
	}
	return strings.TrimSuffix(s, " ")
}

func buildStoreCommand() cli.Command {
	requiredFlags := []string{"name", "input", "db", "storage"}
	usageText := "dibk store " + buildRequiredFlagText(requiredFlags)

	return cli.Command{
		Name:  "store",
		Usage: "Store a version of a binary",
		Flags: append([]cli.Flag{
			cli.StringFlag{Name: "name"},
			cli.StringFlag{Name: "input"},
			cli.IntFlag{Name: "mbperblock", Value: 10},
		}, getCommonSubcommandFlags()...),
		SkipFlagParsing: false,
		HideHelp:        false,
		Hidden:          false,
		UsageText:       usageText,
		Action: func(c *cli.Context) error {
			for _, flag := range requiredFlags {
				if !c.IsSet(flag) {
					err := fmt.Errorf("Required option \"%s\" is missing", flag)
					fmt.Println(err)
					fmt.Println("Usage: " + usageText)
					return err
				}
			}

			err := store(c)
			if err != nil {
				fmt.Printf("Error = %v\n", err)
				fmt.Println("Usage: " + usageText)
			}
			return err
		},
	}
}

func buildRetrieveCommand() cli.Command {
	requiredFlags := []string{"name", "output", "version", "db", "storage"}
	usageText := "dibk retrieve " + buildRequiredFlagText(requiredFlags)

	return cli.Command{
		Name:  "retrieve",
		Usage: "Retrieve a version of a binary",
		Flags: append([]cli.Flag{
			cli.StringFlag{Name: "name"},
			cli.IntFlag{Name: "version"},
			cli.StringFlag{Name: "output"},
		}, getCommonSubcommandFlags()...),
		SkipFlagParsing: false,
		HideHelp:        false,
		Hidden:          false,
		Action: func(c *cli.Context) error {
			for _, flag := range requiredFlags {
				if !c.IsSet(flag) {
					err := fmt.Errorf("Required option \"%s\" is missing", flag)
					fmt.Println(err)
					fmt.Println("Usage: " + usageText)
					return err
				}
			}

			err := retrieve(c)
			if err != nil {
				fmt.Printf("Error = %v\n", err)
				fmt.Println("Usage: " + usageText)
			}
			return err
		},
	}
}
