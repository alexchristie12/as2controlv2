package main

import (
	"as2controlv2/config"
	"as2controlv2/db"
	"as2controlv2/weather"
	"fmt"
	"os"
)

func main() {

	// Take in a config.json as the config, and then load it, the argument
	// to this program is the config file

	// as2controlv2 run <config-file>

	if len(os.Args) == 2 && os.Args[1] == "example" {
		// Print out the example config
		bytes, err := config.MarshalExampleConfig(config.MakeExampleConfig())
		if err != nil {
			fmt.Println("Could not create the example config")
			os.Exit(1)
		}
		fmt.Println(string(bytes))
		os.Exit(0)
	}

	fmt.Println("CC3501 Irrigation control system central control")
	if len(os.Args) != 3 {
		// Print the debug information
		fmt.Println("as2controlv2 run <config-file.json>")
		fmt.Println("Please give this a config file in this format")
		fmt.Println("Please run: 'as2controlv2 example' for an example config file")
		os.Exit(1)
	}

	if len(os.Args) == 3 && os.Args[1] == "run" {
		// Load the config
		bytes, err := os.ReadFile(os.Args[2])
		if err != nil {
			fmt.Println("Could not load config file")
			os.Exit(1)
		}
		// Load the config
		conf, err := config.LoadConfig(bytes)
		if err != nil {
			fmt.Println("Invalid config file")
		}

		// Load up the Datbase connection
		dbHandler, err := db.DBInit(conf.DatabaseConfig)
		if err != nil {
			fmt.Println("Error loading database config: ", err.Error())
			os.Exit(1)
		}

		// Load up the open weather map connection
		weatherHandler := weather.WeatherInit(conf.WeatherAPIConfig)

		// Load the serial connection
	}
}
