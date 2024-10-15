package control

import (
	"as2controlv2/config"
	"as2controlv2/db"
	"as2controlv2/serial"
	"as2controlv2/weather"
)

type ControlSystem struct {
	systemConfig config.Config // We want to be able to access all of our config

	dbHandler      db.DBConnection         // Connection to write to InfluxDB
	weatherHandler weather.WeatherAPI      // Connection to pull data from OpenWeatherMap
	serialHandler  serial.SerialConnection // Connection to the serial port (Bluetooth module)
}
