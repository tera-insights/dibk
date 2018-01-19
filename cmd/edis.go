package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tera-insights/edis"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/urfave/cli"
)

func main() {
	app := buildApp()
	app.Run(os.Args)
}

func buildApp() *cli.App {
	app := cli.NewApp()
	app.Description = "Encrypted Disk Image Storage"
	app.Name = "edis"
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
	e, err := makeEngineFromContext(c)
	if err != nil {
		return err
	}

	if c.Bool("latest") {
		return e.RetrieveLatestVersionOfObject(c.String("output"), c.String("name"))
	}
	return e.RetrieveObject(c.String("output"), c.String("name"), c.Int("version"))
}

func makeEngineFromContext(c *cli.Context) (edis.Engine, error) {
	return edis.MakeEngine(edis.Configuration{
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

func getCommonSubcommandFlags() []cli.Flag {
	return []cli.Flag{
		cli.BoolFlag{Name: "directio", Usage: "If enabled, use directIO to read and write files"},
		cli.StringFlag{Name: "db", Usage: "Path to the SQLite3 database that holds metadata about the backups"},
		cli.StringFlag{Name: "storage", Usage: "Path to the directory to use for storage"},
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
	usageText := "edis store " + buildRequiredFlagText(requiredFlags)

	return cli.Command{
		Name:  "store",
		Usage: "Store a version of a binary",
		Flags: append([]cli.Flag{
			cli.StringFlag{Name: "name", Usage: "The name of the object to store"},
			cli.StringFlag{Name: "input", Usage: "Path to the file to read"},
			cli.IntFlag{Name: "mbperblock", Value: 10, Usage: "How many megabytes are in a block. Must be an integer"},
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
	requiredFlags := []string{"name", "output", "db", "storage"}
	usageText := "\nedis retrieve --latest " + buildRequiredFlagText(requiredFlags) + "\nedis retrieve --version $VERSION " + buildRequiredFlagText(requiredFlags)

	return cli.Command{
		Name:      "retrieve",
		Usage:     "Retrieve a version of a binary",
		UsageText: usageText,
		Flags: append([]cli.Flag{
			cli.StringFlag{Name: "name", Usage: "The name of the object to retrieve"},
			cli.IntFlag{Name: "version", Value: 1, Usage: "Specify an object version to retrieve. Either this or --latest must be set"},
			cli.StringFlag{Name: "output", Usage: "Path into which the retrieved object should be written"},
			cli.BoolFlag{Name: "latest", Usage: "If enabled, fetch the latest version. Either this or --version must be set"},
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

			if !c.IsSet("latest") && !c.IsSet("version") {
				err := fmt.Errorf("Neither \"latest\" nor \"version\" was set")
				fmt.Println(err)
				fmt.Println("Usage: " + usageText)
				return err
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
