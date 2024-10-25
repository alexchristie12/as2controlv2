package control

import (
	"as2controlv2/config"
	"as2controlv2/db"
	"as2controlv2/serial"
	"as2controlv2/weather"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// Overall struct for the control system
type ControlSystem struct {
	logger *slog.Logger

	systemConfig config.Config // We want to be able to access all of our config

	dbHandler             db.DBConnection              // Connection to write to InfluxDB
	weatherHandler        weather.WeatherAPI           // Connection to pull data from OpenWeatherMap
	serialHandler         serial.SerialConnection      // Connection to the serial port (Bluetooth module)
	currentWeatherValues  weather.CurrentWeatherResult // The current weather prediction
	currentSensorAverages []db.CurrentLocalValues      // The current average from all the sensor readings
	systemTiming          Timings                      // The time coordination for the whole system
}

// Struct to store the timings for the system
type Timings struct {
	NextRemoteUnitFetchTime    time.Time          // The next time to fetch data from the remote units
	NextWeatherReportFetchTime time.Time          // The next time to fetch weather data from open weather map
	NextRainReportFetchTime    time.Time          // The next time to fetch rain data from open weather map
	NextWateringTime           map[uint]time.Time // In this case the keys are the corresponding zone, deleted after we are done
	WateringUntilTime          map[uint]time.Time // This is where we store the time we water until, deleted after we are done
}

// Initialise the control system
func ControlSystemInit(logger *slog.Logger, config config.Config, dbHandler db.DBConnection, weatherHandler weather.WeatherAPI, serialHandler serial.SerialConnection) *ControlSystem {
	return &ControlSystem{
		systemConfig:          config,
		logger:                logger,
		dbHandler:             dbHandler,
		weatherHandler:        weatherHandler,
		serialHandler:         serialHandler,
		currentWeatherValues:  weather.CurrentWeatherResult{},
		currentSensorAverages: make([]db.CurrentLocalValues, len(config.RemoteUnitConfigs)),
		systemTiming:          makeTimings(),
	}
}

func makeTimings() Timings {
	return Timings{
		NextRemoteUnitFetchTime:    time.Now(),
		NextWeatherReportFetchTime: time.Now(),
		NextRainReportFetchTime:    time.Now(),
		NextWateringTime:           make(map[uint]time.Time),
	}
}

// Fetch the readings from a single remote unit. NOTE: This function is probably too long
func (cs *ControlSystem) FetchRemoteUnitReading(rmu config.RemoteUnitConfig, currentValues *db.CurrentLocalValues) error {
	readings, err := cs.serialHandler.PollDevice(rmu.UnitNumber)
	cs.logger.Debug(fmt.Sprint(readings))
	if err != nil {
		cs.logger.Error(fmt.Sprintf("could not get readings from device zone id: %d", rmu.UnitNumber))
		return err
	}

	// We don't necessarily want to exit now
	hardwareID := readings[0].Value
	if readings[0].Name != "hardware_id" {
		cs.logger.Warn("hardware ID is not the first value sent in poll, please validate data link integrity")
	}

	if hardwareID != float64(rmu.UnitNumber) {
		cs.logger.Warn("hardware ID does not match one specified in poll, please check bluetooth link configuration")
	}

	// Check the temperatures
	temperaturesSum := 0.0
	temperatureCount := 0
	humiditySum := 0.0
	humidityCount := 0
	soilMositureSum := 0.0
	soilMoistureCount := 0
	flowRateSum := 0.0
	flowRateCount := 0
	// Since there is only one water on, we can just look for the water_on

	cs.logger.Info(fmt.Sprintf("got %d readings", len(readings)))
	for _, r := range readings {
		if len(readings) != 14 {
			cs.logger.Error("invalid number of entries")
			return errors.New("invalid number of entries")
		}
		if strings.Contains(strings.ToLower(r.Name), "temperature") {
			if r.Value > 9000 {
				// Its over 9000!!!
				continue
			}
			temperatureCount++
			temperaturesSum += r.Value
			continue
		}

		if strings.Contains(strings.ToLower(r.Name), "humidity") {
			if r.Value > 9000 {
				// Its over 9000!!!
				continue
			}
			humidityCount++
			humiditySum += r.Value
			continue
		}

		if strings.Contains(strings.ToLower(r.Name), "soil_moisture") {
			if r.Value > 9000 {
				// Its over 9000!!!
				continue
			}
			soilMoistureCount++
			soilMositureSum += r.Value
			continue
		}

		if strings.Contains(strings.ToLower(r.Name), "flow_rate") {
			if r.Value > 9000 {
				// It's over 9000!
				continue
			}
			flowRateCount++
			flowRateSum += r.Value
			continue
		}
	}

	// Check that the last one is flow rate
	lastReading := readings[len(readings)-1]

	if !strings.Contains(lastReading.Name, "water_on") {
		cs.logger.Warn("water on is not the last value in poll, please verify data integrity")
		currentValues.WaterOn = lastReading.Value
	}

	currentValues.Temperature = temperaturesSum / float64(temperatureCount)
	currentValues.Humidity = humiditySum / float64(humidityCount)
	currentValues.SoilMoisture = soilMositureSum / float64(soilMoistureCount)
	// Write these to InfluxDB
	// Make the tags
	tags := db.Tags{
		SystemName:     cs.systemConfig.Name,
		RemoteUnitName: rmu.UnitName,
	}
	err = cs.dbHandler.WriteUnitMetrics(rmu.UnitName, *currentValues, tags)
	if err != nil {
		return err
	}

	// TODO: Might want better error handling here, this function is really lloooonnnnngggg.

	// Unfortunately, we need to add more to this function, because we will handle the watering task here
	if cs.checkIfNeedsWateringOn(rmu.UnitNumber) {
		if err = cs.HandleWateringOnEvent(rmu.UnitNumber); err != nil {
			cs.logger.Info(fmt.Sprintf("failed to turn watering on for unit %d", rmu.UnitNumber))
		} else {
			cs.logger.Info(fmt.Sprintf("turned watering on for unit %d", rmu.UnitNumber))
		}
	}

	if cs.checkIfNeedsWateringOff(rmu.UnitNumber) {
		if err = cs.HandleWateringOffEvent(rmu.UnitNumber); err != nil {
			cs.logger.Info(fmt.Sprintf("failed to turn watering off for unit %d", rmu.UnitNumber))
		} else {
			cs.logger.Info(fmt.Sprintf("turned watering off for unit %d", rmu.UnitNumber))
		}
	}
	cs.logger.Info(fmt.Sprintf("fetched data from zone %d", rmu.UnitNumber))
	return nil
}

// Fetch all of the currently configured remote units
func (cs *ControlSystem) FetchRemoteUnitReadings() error {
	cs.logger.Info(fmt.Sprintf("current unit is: %d", cs.serialHandler.CurrentDevice))
	// For each remote unit, grab all the values
	for i, rmu := range cs.systemConfig.RemoteUnitConfigs {
		// If check that we are on the right connection
		if rmu.UnitNumber != cs.serialHandler.CurrentDevice {
			// Switch to the device we want data from
			cs.logger.Info(fmt.Sprintf("attempting to switch from device %d to device %d", cs.serialHandler.CurrentDevice, rmu.UnitNumber))
			if err := cs.serialHandler.SwitchDevice(rmu.UnitNumber); err != nil {
				return err
			}
		}
		err := cs.FetchRemoteUnitReading(rmu, &cs.currentSensorAverages[i])
		if err != nil {
			// Don't return, just move on
			cs.logger.Error(fmt.Sprintf("could not fetch sensor data from unit %d: %s", rmu.UnitNumber, err.Error()))
			continue
		}
	}
	// Now go back to the original device
	cs.serialHandler.CurrentDevice = 0
	return nil
}

// Fetch the current weather data from OpenWeatherMap
func (cs *ControlSystem) FetchWeatherData() error {
	weatherResult, err := cs.weatherHandler.GetCurrentWeather()
	if err != nil {
		cs.logger.Error("could not fetch weather data")
		return err
	}

	// Put it into the control base
	cs.currentWeatherValues = weatherResult
	cs.logger.Info("fetched weather data")
	// Write to influxDB
	if err := cs.dbHandler.WriteWeatherMetrics(weatherResult); err != nil {
		return err
	}
	return nil
}

// Check that we need to water, for each zone, and scedule watering
func (cs *ControlSystem) CheckWatering() {
	/*
		Check the current soil moisture in each zone, if it below the required threshold,
		start watering in 20 minutes, unless the cancel endpoint is hit...

		- This threshold is 25% soil moisture
	*/
	for i, rmu := range cs.currentSensorAverages {
		if rmu.SoilMoisture < 25 {
			if _, ok := cs.systemTiming.NextWateringTime[cs.systemConfig.RemoteUnitConfigs[i].UnitNumber]; ok {
				continue
			}
			if _, ok := cs.systemTiming.WateringUntilTime[cs.systemConfig.RemoteUnitConfigs[i].UnitNumber]; ok {
				continue
			}
			if cs.systemConfig.Mode == "automatic" {
				// Set the watering to go off in 20 minutes
				cs.systemTiming.NextWateringTime[cs.systemConfig.RemoteUnitConfigs[i].UnitNumber] = time.Now()
				cs.logger.Info(fmt.Sprintf("scheduling unit number %d for watering in one minute", cs.systemConfig.RemoteUnitConfigs[i].UnitNumber))
			} else if cs.systemConfig.Mode == "manual" {
				// Just suggest that we water, send shit to Grafana
				// Work out how I am going to send off the warnings
			}
		}
	}
}

// Handle watering a particular zone
func (cs *ControlSystem) HandleWateringOnEvent(unitNumber uint) error {
	// We need to be in the right connection to do this right. So we do this when we
	// get the sensor readings
	if cs.systemTiming.WateringUntilTime == nil {
		cs.systemTiming.WateringUntilTime = make(map[uint]time.Time, 0)
	}
	if cs.systemTiming.NextWateringTime == nil {
		cs.systemTiming.NextWateringTime = make(map[uint]time.Time, 0)
	}
	err := cs.serialHandler.WriteToDevice(fmt.Sprintf("water_on=%d\r\n", unitNumber))
	if err != nil {
		return err
	}
	// Then delete it from the map as we don't need to store it anymore
	cs.logger.Info(fmt.Sprintf("turning on watering for unit %d", unitNumber))
	delete(cs.systemTiming.NextWateringTime, unitNumber)
	// Then set a timer to water for some amount of time, maybe 20 minutes
	cs.systemTiming.WateringUntilTime[unitNumber] = time.Now().Add(30 * time.Second)
	return nil
}

// Check if we need to turn watering on for a given unit
func (cs *ControlSystem) checkIfNeedsWateringOn(unitNumber uint) bool {
	// Check if it is to be watered now
	if val, ok := cs.systemTiming.NextWateringTime[unitNumber]; !ok {
		return false // It is not in the timings
	} else if time.Now().After(val) {
		return true // It is time to water it now
	} else {
		return false // It is not time yet
	}
}

// Check if we need to turn watering off for a given unit
func (cs *ControlSystem) checkIfNeedsWateringOff(unitNumber uint) bool {
	if val, ok := cs.systemTiming.WateringUntilTime[unitNumber]; !ok {
		return false // We are not watering at the moment
	} else if time.Now().After(val) {
		return true // It is time to turn the watering off
	} else {
		return false // It is not yet time to turn the watering off
	}
}

// Handle turning off watering for a particular device
func (cs *ControlSystem) HandleWateringOffEvent(unitNumber uint) error {
	if cs.systemTiming.WateringUntilTime == nil {
		cs.systemTiming.WateringUntilTime = make(map[uint]time.Time, 0)
	}
	if cs.systemTiming.NextWateringTime == nil {
		cs.systemTiming.NextWateringTime = make(map[uint]time.Time, 0)
	}
	err := cs.serialHandler.WriteToDevice(fmt.Sprintf("water_off=%d\r\n", unitNumber))
	if err != nil {
		return err
	}
	cs.logger.Info(fmt.Sprintf("turning off watering for unit %d", unitNumber))
	// Now delete it from the watering map
	delete(cs.systemTiming.WateringUntilTime, unitNumber)
	return nil
}

// Collection of sensor warnings, would be better as a type warnings []warning
type warnings struct {
	sensorWarnings []warning
}

type warning struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Msg   string  `json:"message"`
}

// Gets the warnings for the system, based on most current data
func (cs *ControlSystem) GetNewWarnings() warnings {
	/*
		In this function we are going to check for any warning, especially
		relating to the sensor readings (temperature and humidity), and the
		data from the weather (mainly temperature as well), see what I want
		to use
	*/
	ws := make([]warning, 0)

	// Check temperatures, threshold is above 35 degrees, below 12 degrees
	ws = append(ws, cs.generateTemperatureSensorWarnings()...)

	// Check humidities, threshold is above 90%, below 20%
	ws = append(ws, cs.generateHumiditySensorWarnings()...)

	// Check weather, if the temperatures and humidities are the same, lots of rain (>100mm),
	// or lots of cloud cover
	return warnings{
		sensorWarnings: ws,
	}
}

// Generate temperature sensor warnings from the last remote unit poll
func (cs *ControlSystem) generateTemperatureSensorWarnings() []warning {
	ws := make([]warning, 0)
	for i, v := range cs.currentSensorAverages {
		if v.Temperature > 35 {
			ws = append(ws, warning{
				Name:  cs.systemConfig.RemoteUnitConfigs[i].UnitName,
				Value: v.Temperature,
				Msg:   fmt.Sprintf("Temperture is high, Zone %d should be monitored", i),
			})
		} else if v.Temperature < 12 {
			ws = append(ws, warning{
				Name:  cs.systemConfig.RemoteUnitConfigs[i].UnitName,
				Value: v.Temperature,
				Msg:   fmt.Sprintf("Temperature is low, Zone %d should be monitored", i),
			})
		}

	}
	return ws
}

// Generate humidity warnings from the last set of remote unit poll.
func (cs *ControlSystem) generateHumiditySensorWarnings() []warning {
	ws := make([]warning, 0)
	for i, v := range cs.currentSensorAverages {
		if v.Humidity > 90 {
			ws = append(ws, warning{
				Name:  cs.systemConfig.RemoteUnitConfigs[i].UnitName,
				Value: v.Humidity,
				Msg:   fmt.Sprintf("Humidity is very high, Zone %d should be monitored", i),
			})
		} else if v.Humidity < 20 {
			ws = append(ws, warning{
				Name:  cs.systemConfig.RemoteUnitConfigs[i].UnitName,
				Value: v.Humidity,
				Msg:   fmt.Sprintf("Humidity is very low, Zone %d should be monitored", i),
			})
		}
	}
	return ws
}

// Generate weather warnings from the last weather data fetch
func (cs *ControlSystem) generateWeatherWarnings() []warning {
	// Check the cloud cover mainly,
	ws := make([]warning, 0)
	if cs.currentWeatherValues.Clouds.All > 90 {
		ws = append(ws, warning{
			Name:  "Cloud cover",
			Value: cs.currentWeatherValues.Clouds.All,
			Msg:   "It is very cloudy, plants may not receive optimal sunlight",
		})
	}

	// Check the wind, 20m/s is nearly cyclonic
	if cs.currentWeatherValues.Wind.Speed > 20 {
		ws = append(ws, warning{
			Name:  "High Wind Speed",
			Value: cs.currentWeatherValues.Wind.Speed,
			Msg:   "The current wind speed is very high, ensure plants are sheltered",
		})
	}
	return ws
}

// This is the main event loop for the system, where it checks the timings for everything,
// triggers the appropriate event
func (cs *ControlSystem) CheckTimings() error {
	for {
		// Check fetch times
		if err := cs.CheckFetchTimes(); err != nil {
			cs.logger.Error(fmt.Sprintf("could not fetch remote data: %s", err.Error()))
		}

		// Check weather times
		if err := cs.CheckWeatherFetchTimes(); err != nil {
			cs.logger.Error(fmt.Sprintf("could not fetch weather data: %s", err.Error()))
		}

		// Check that we need to water soon
		cs.CheckWatering()

	}
}

// Check the fetch timing for the system
func (cs *ControlSystem) CheckFetchTimes() error {
	if time.Now().After(cs.systemTiming.NextRemoteUnitFetchTime) {
		// Set the next time
		cs.systemTiming.NextRemoteUnitFetchTime = time.Now().Add(time.Duration(cs.systemConfig.RemoteIntervalSeconds) * time.Second)
		// Fetch the new remote units
		err := cs.FetchRemoteUnitReadings()
		if err != nil {
			return err
		}
	}
	return nil
}

// Check the weather fetching times for the system
func (cs *ControlSystem) CheckWeatherFetchTimes() error {
	if time.Now().After(cs.systemTiming.NextWeatherReportFetchTime) {
		// Set the next time
		cs.systemTiming.NextWeatherReportFetchTime = time.Now().Add(time.Duration(cs.systemConfig.WeatherIntervalSeconds) * time.Second)
		if err := cs.FetchWeatherData(); err != nil {
			return err
		}
	}
	return nil
}

// Check if we need to water for each system
func (cs *ControlSystem) CheckWateringOnTimes() error {
	if cs.systemTiming.NextWateringTime == nil {
		cs.systemTiming.NextWateringTime = make(map[uint]time.Time)
	}
	if cs.systemTiming.WateringUntilTime == nil {
		cs.systemTiming.WateringUntilTime = make(map[uint]time.Time)
	}
	if len(cs.systemTiming.NextWateringTime) == 0 {
		return nil
	}
	// Now check the timing
	for devID, wateringTime := range cs.systemTiming.NextWateringTime {
		if time.Now().After(wateringTime) {
			// Trigger the watering
			if err := cs.HandleWateringOnEvent(devID); err != nil {
				return err
			}
		}
	}
	return nil
}

// Check if we need to turn the watering off for a remote unit
func (cs *ControlSystem) CheckWateringOffTimes() error {
	if len(cs.systemTiming.WateringUntilTime) == 0 {
		return nil
	}
	// Now check the timing
	for devID, wateringOffTime := range cs.systemTiming.WateringUntilTime {
		if time.Now().After(wateringOffTime) {
			// Trigger the watering off event
			if err := cs.HandleWateringOffEvent(devID); err != nil {
				return err
			}
		}
	}
	return nil
}
