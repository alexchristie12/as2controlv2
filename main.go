package main

import (
	"as2controlv2/config"
	"as2controlv2/control"
	"as2controlv2/db"
	"as2controlv2/serial"
	"as2controlv2/weather"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
)

func CheckArgs(args []string) error {
	if len(args) == 1 {
		fmt.Println("as2controlv2 run <config file> | example | help")
		return errors.New("no args given")
	}
	if args[1] != "run" && args[1] != "example" && args[1] != "help" {
		return errors.New("invalid use of program, valid args are 'run', 'example', or 'help'")
	}

	if args[1] == "run" && len(args) != 3 {
		return errors.New("invalid use of 'run' command, please provide a config file")
	} else if len(args) == 3 {
		// Check that the file exists
		if _, err := os.Stat(args[2]); err != os.ErrExist {
			return err
		}
	}

	return nil
}

func LoadConfig(fileName string) (config.Config, error) {
	bytes, err := os.ReadFile(fileName)
	if err != nil {
		return config.Config{}, err
	}
	conf, err := config.LoadConfig(bytes)
	if err != nil {
		return config.Config{}, err
	}
	return conf, nil

}

func HandleExampleConfigArg() {
	conf := config.MakeExampleConfig()
	bytes, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		fmt.Println("could not make example config")
		os.Exit(1)
	}
	fmt.Print(string(bytes))
}

func HandleHelpArg() {
	fmt.Println(`as2controlv2 - Irrigation system control system
args:
		- help: Print the help information of the system
		- example: print an example config to standard output
		- run <config-file>: run the control system with the given config file`)
}

func SetupRoutes(r *gin.Engine, cs *control.ControlSystem) {
	r.GET("/api/warnings", cs.RouteGETWarnings)
	r.POST("/api/delay", cs.RoutePOSTDelayWatering)
	r.POST("/api/cancel", cs.RoutePOSTCancelWatering)
	r.POST("/api/water-now", cs.RoutePOSTWaterNow)
}

func main() {

	// Take in a config.json as the config, and then load it, the argument
	// to this program is the config file

	// Parse all the flags
	if err := CheckArgs(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if os.Args[1] == "example" {
		HandleExampleConfigArg()
		os.Exit(0)
	}

	if os.Args[1] == "help" {
		HandleHelpArg()
		os.Exit(0)
	}

	if os.Args[1] != "run" {
		os.Exit(0)
	}

	// Load the config file
	conf, err := LoadConfig(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	// conf := config.MakeTestingConfig()
	for _, v := range conf.RemoteUnitConfigs {
		fmt.Println("Got unit number ", v.UnitNumber)
	}

	// as2controlv2 run <config-file>
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	// Load up the Datbase connection
	dbHandler, err := db.DBInit(conf.DatabaseConfig)
	if err != nil {
		fmt.Println("Error loading database config: ", err.Error())
		os.Exit(1)
	}

	// Load up the open weather map connection
	weatherHandler := weather.WeatherInit(conf.WeatherAPIConfig)

	// Load the serial connection
	serialHandler, err := serial.SerialConnectionInit(conf.SerialConfig, logger)
	if err != nil {
		fmt.Println("Error loading serial connection config: ", err.Error())
	}

	// Now enter the loop, spawn each process in a separate thread
	controller := control.ControlSystemInit(logger, conf, dbHandler, weatherHandler, serialHandler)
	// Spawn the server on a different thread
	// Define all the HTTP routes
	gin.DisableConsoleColor()
	f, err := os.Create("gin.log")
	if err != nil {
		log.Println("Could not create log file for gin")
	}
	gin.DefaultWriter = io.MultiWriter(f)
	r := gin.Default()
	SetupRoutes(r, controller)
	go func() { // This runs this function asyncronously, so we can sit in our main loop
		log.Fatal(r.Run())
	}()

	for {
		// err := controller.FetchRemoteUnitReadings()
		// if err != nil {
		// 	fmt.Println("Error fetching remote unit readings: ", err.Error())
		// }
		// time.Sleep(15 * time.Second)
		err := controller.CheckTimings()
		if err != nil {
			// This should never be reached
			os.Exit(1)
		}
	}
}
