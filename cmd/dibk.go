package main

import (
	"dibk"
	"encoding/json"
	"os"
)

func main() {
	conf, err := readConfig()
	if err != nil {
		panic(err)
	}

	e, err := dibk.MakeEngine(conf)
	if err != nil {
		panic(err)
	}
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
