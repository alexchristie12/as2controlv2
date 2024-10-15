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

type CurrentLocalValues struct {
	avgTemperature  float64
	avgHumidity     float64
	avgSoilMoisture float64
}

func (cs *ControlSystem) FetchRemoteUnitReadings() error {
	// For each remote unit, grab all the values
	for _, rmu := range cs.systemConfig.RemoteUnitConfigs {
		// If this fails we should retry a maximum of three times.
		readings, err := cs.serialHandler.PollDevice(rmu.UnitNumber)
		if err != nil {
			return err
		}
		// Send off the readings to InfluxDB
		err = cs.dbHandler.WriteSensorReadings(readings, rmu, cs.systemConfig.Name)
		if err != nil {
			// TODO: This needs to be in a separate function, we can have it failing for two
			// different reasons!!
			return err
		}
		// Pull out the data that we want from each, we will need to know what relates to what
		// Average it out where required.
	}

	// This is when everything was been performed correctly
	return nil
}

func (cs *ControlSystem) FetchWeatherData() error {
	weatherResult, err := cs.weatherHandler.GetCurrentWeather()
	if err != nil {
		return err
	}
	// Set the control system based on the weather results
	// Come up with some recomendations

	return nil
}

func (cs *ControlSystem) FetchRainData() error {
	rainResult, err := cs.weatherHandler.GetYesterdaysRain()
	if err != nil {
		return err
	}
	// Send of this data and make recommendations based on that

	return nil
}
