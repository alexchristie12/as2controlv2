package control

import (
	"as2controlv2/config"
	"as2controlv2/db"
	"as2controlv2/serial"
	"as2controlv2/weather"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

type ControlSystem struct {
	logger slog.Logger

	systemConfig config.Config // We want to be able to access all of our config

	dbHandler             db.DBConnection              // Connection to write to InfluxDB
	weatherHandler        weather.WeatherAPI           // Connection to pull data from OpenWeatherMap
	serialHandler         serial.SerialConnection      // Connection to the serial port (Bluetooth module)
	currentWeatherValues  weather.CurrentWeatherResult // The current weather prediction
	yesterdaysRainValue   weather.RainResult           // The rain from the previous day
	currentSensorAverages []CurrentLocalValues         // The current average from all the sensor readings
	wateringStats         []WateringStats              // Store information on whether we need to water a particular zone
}

type CurrentLocalValues struct {
	Temperature  float64
	Humidity     float64
	SoilMoisture float64
	FlowRate     float64
}

type CurrentPrediction struct {
}

type RainStats struct {
	TimeSinceLastRain time.Time
}

type WateringStats struct {
	ZoneID             uint      // The corresponding zone ID
	TimeSinceLastWater time.Time // The last time the lawn was watered
	TimeToNextWater    time.Time // The time until we should water again
}

func ControlSystemInit(logger slog.Logger, config config.Config, dbHandler db.DBConnection, weatherHandler weather.WeatherAPI, serialHandler serial.SerialConnection) *ControlSystem {
	return &ControlSystem{
		systemConfig:          config,
		logger:                logger,
		dbHandler:             dbHandler,
		weatherHandler:        weatherHandler,
		serialHandler:         serialHandler,
		currentWeatherValues:  weather.CurrentWeatherResult{},
		yesterdaysRainValue:   weather.RainResult{},
		currentSensorAverages: make([]CurrentLocalValues, len(config.RemoteUnitConfigs)),
		wateringStats:         makeWateringStats(config.RemoteUnitConfigs),
	}
}

func makeWateringStats(configs []config.RemoteUnitConfig) []WateringStats {
	wateringStats := make([]WateringStats, len(configs))
	for i, conf := range wateringStats {
		wateringStats[i].ZoneID = conf.ZoneID
	}
	return wateringStats
}

func (cs *ControlSystem) FetchRemoteUnitReading(rmu config.RemoteUnitConfig, currentValues *CurrentLocalValues) error {
	readings, err := cs.serialHandler.PollDevice(rmu.UnitNumber)
	if err != nil {
		cs.logger.Error(fmt.Sprintf("could not get readings from device zone id: %d", rmu.UnitNumber))
		return err
	}
	// Send off the readings to influxDB
	err = cs.dbHandler.WriteSensorReadings(readings, rmu, cs.systemConfig.Name)
	if err != nil {
		cs.logger.Error(fmt.Sprintf("could not write readings for device zone id %d to influxDB: %e", rmu.UnitNumber, err))
		return err
	}
	// Now look for values, pray that we have the right values, first value must be the zone id, last must be the flow rate
	// These act as integrity checks

	// We don't necessarily want to exit now
	hardwareID := readings[0].Value
	if readings[0].Name != "hardware_id" {
		cs.logger.Warn("hardware ID is not the first value sent in poll, please validate data link integrity")
	}

	if hardwareID != float32(rmu.UnitNumber) {
		cs.logger.Warn("hardware ID does not match one specified in poll, please check bluetooth link configuration")
	}

	// Check the temperatures
	temperaturesSum := 0.0
	temperatureCount := 0
	humiditySum := 0.0
	humidityCount := 0
	soilMositureSum := 0.0
	soilMoistureCount := 0

	for _, r := range readings {
		if strings.Contains(strings.ToLower(r.Name), "temperature") {
			temperatureCount++
			temperaturesSum += float64(r.Value)
			continue
		}

		if strings.Contains(strings.ToLower(r.Name), "humidity") {
			humidityCount++
			humiditySum += float64(r.Value)
			continue
		}

		if strings.Contains(strings.ToLower(r.Name), "soil_moisture") {
			soilMoistureCount++
			soilMositureSum += float64(r.Value)
			continue
		}
	}

	// Check that the last one is flow rate
	lastReading := readings[len(readings)-1]

	if lastReading.Name != "flow_rate" {
		cs.logger.Warn("flow rate is not last value in poll, please verify data integrity")
		currentValues.FlowRate = float64(lastReading.Value)
	}

	currentValues.Temperature = temperaturesSum / float64(temperatureCount)
	currentValues.Humidity = humiditySum / float64(humidityCount)
	currentValues.SoilMoisture = soilMositureSum / float64(soilMoistureCount)

	// TODO: Might want better error handling here
	cs.logger.Info(fmt.Sprintf("fetched data from zone %d", rmu.UnitNumber))
	return nil
}

func (cs *ControlSystem) FetchRemoteUnitReadings() error {
	// For each remote unit, grab all the values
	for i, rmu := range cs.systemConfig.RemoteUnitConfigs {
		err := cs.FetchRemoteUnitReading(rmu, &cs.currentSensorAverages[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func (cs *ControlSystem) FetchWeatherData() error {
	weatherResult, err := cs.weatherHandler.GetCurrentWeather()
	if err != nil {
		cs.logger.Error("could not fetch weather data")
		return err
	}
	// Set the control system based on the weather results
	// Come up with some recommendations if we have three mm or rain or
	// more, we can delay our next watering session by a day, if we get
	// 9mm or rain, we can delay it by two days

	// Put it into the control base
	cs.currentWeatherValues = weatherResult
	cs.logger.Info("fetched weather data")
	// Write to influxDB
	return nil
}

func (cs *ControlSystem) FetchRainData() error {
	rainResult, err := cs.weatherHandler.GetYesterdaysRain()
	if err != nil {
		return err
	}
	// Send of this data and make recommendations based on that
	// I think that if we got 4mm of rain that day, we don't need
	// to water the grass
	cs.yesterdaysRainValue = rainResult
	// Write to influxDB

	cs.logger.Info("fetched rain data")
	return nil
}

// Check that we need to water, for each zone, and for how long
func (cs *ControlSystem) CheckWatering() {
	/*
		Check the current soil moisture in each zone, if it below the required threshold,
		start watering in 20 minutes, unless the cancel endpoint is hit...

		- This threshold is 25% soil moisture
	*/
	for i, rmu := range cs.currentSensorAverages {
		if rmu.SoilMoisture < 25 {
			// Trigger a watering event for that particular remote unit, then set the 20 minute timer
			// callback, that is cancelled by an endpoint at /-/cancel?id=x
			if cs.systemConfig.Mode == "automatic" {
				// Spawn the timer, with the deadline
			} else if cs.systemConfig.Mode == "manual" {
				// Just suggest that we water, send shit to Grafana
			}
		}
	}
}

// Check if there is any environmental issues, temperature, humidity mainly
func (cs *ControlSystem) CheckForEnvironmentalIssues() {

}
