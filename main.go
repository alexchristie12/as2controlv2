package main

import (
	"as2controlv2/config"
	"as2controlv2/control"
	"as2controlv2/db"
	"as2controlv2/serial"
	"as2controlv2/weather"
	"fmt"
	"log/slog"
	"os"
	"time"
)

func main() {

	// Take in a config.json as the config, and then load it, the argument
	// to this program is the config file

	// as2controlv2 run <config-file>
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	conf := config.MakeTestingConfig()

	// Load up the Datbase connection
	dbHandler, err := db.DBInit(conf.DatabaseConfig)
	if err != nil {
		fmt.Println("Error loading database config: ", err.Error())
		os.Exit(1)
	}

	// Load up the open weather map connection
	weatherHandler := weather.WeatherInit(conf.WeatherAPIConfig)

	// Load the serial connection
	serialHandler, err := serial.SerialConnectionInit(conf.SerialConfig)
	if err != nil {
		fmt.Println("Error loading serial connection config: ", err.Error())
	}

	// Now enter the loop, spawn each process in a separate thread
	controller := control.ControlSystemInit(logger, conf, dbHandler, weatherHandler, serialHandler)
	for {
		err := controller.FetchRemoteUnitReadings()
		if err != nil {
			fmt.Println("Error fetching remote unit readings: ", err.Error())
		}
		time.Sleep(15 * time.Second)
	}
}
