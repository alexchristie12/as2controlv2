package config

import (
	"encoding/json"
)

// Struct that stores the configuration for the control system
type Config struct {
	Name                   string               `json:"name"`
	Mode                   string               `json:"mode"` // This can be either automatic, or manual
	WeatherIntervalSeconds uint                 `json:"weather_scrape_interval"`
	RemoteIntervalSeconds  uint                 `json:"remote_interval_seconds"`
	SerialConfig           SerialConfig         `json:"serial_config"`
	DatabaseConfig         InfluxDBConfig       `json:"influxdb_config"`
	WeatherAPIConfig       OpenWeatherMapConfig `json:"weather_api_config"`
	RemoteUnitConfigs      []RemoteUnitConfig   `json:"remote_configs"`
}

type RemoteUnitConfig struct {
	UnitName   string `json:"name"`
	UnitNumber uint   `json:"number"`
}

type InfluxDBConfig struct {
	URL          string `json:"url"`
	Organisation string `json:"string"`
	Bucket       string `json:"bucket"`
	Token        string `json:"token"`
}

type OpenWeatherMapConfig struct {
	URL       string  `json:"url"`
	Token     string  `json:"token"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type SerialConfig struct {
	Port           string `json:"serial_port"`
	BaudRate       uint   `json:"baud_rate"`
	TimeoutSeconds uint   `json:"timeout_sec"`
}

// Make an example configuration, for the example arg
func MakeExampleConfig() Config {
	return Config{
		Mode:                   "automatic",
		Name:                   "example_system_config",
		WeatherIntervalSeconds: 3600,
		RemoteIntervalSeconds:  60,
		SerialConfig: SerialConfig{
			Port:           "/dev/ttyS0",
			BaudRate:       115200,
			TimeoutSeconds: 5,
		},
		DatabaseConfig: InfluxDBConfig{
			URL:          "http://localhost:8086",
			Organisation: "My_Organisation",
			Bucket:       "My_Bucket",
			Token:        "my_super_long_token",
		},
		WeatherAPIConfig: OpenWeatherMapConfig{
			URL:       "http://api.openweathermap.org",
			Token:     "my_super_long_open_weathermap_config",
			Latitude:  -19.2569391,
			Longitude: 146.8239537,
		},
		RemoteUnitConfigs: []RemoteUnitConfig{
			{
				UnitName:   "unit_1",
				UnitNumber: 1,
			},
			{
				UnitName:   "unit_2",
				UnitNumber: 2,
			},
			{
				UnitName:   "unit_3",
				UnitNumber: 3,
			},
			{
				UnitName:   "unit_4",
				UnitNumber: 4,
			},
			{
				UnitName:   "unit_5",
				UnitNumber: 5,
			},
		},
	}
}

// Marshal the example config
func MarshalExampleConfig(conf Config) ([]byte, error) {
	bytes, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

// Load a config
func LoadConfig(bytes []byte) (Config, error) {
	var conf Config
	err := json.Unmarshal(bytes, &conf)
	if err != nil {
		return Config{}, err
	}
	return conf, nil
}

// Make the config used throughout testing
func MakeTestingConfig() Config {
	return Config{
		Name:                   "example_system_config",
		Mode:                   "automatic",
		WeatherIntervalSeconds: 3600,
		RemoteIntervalSeconds:  30,
		SerialConfig: SerialConfig{
			Port:           "/dev/ttyS0",
			BaudRate:       115200,
			TimeoutSeconds: 5,
		},
		DatabaseConfig: InfluxDBConfig{
			URL:          "http://192.168.77.196:8086",
			Organisation: "Water_Monitoring",
			Bucket:       "testing",
			// Token:        "1vCqkEI_vBPdjuoOBrFNI5JA2yIV3C8DnD2C3KyWpgq3XWkdAXM7cHbo2fAxh5jZt2ppFhD2Q1PRoO99bBuArw==",
			Token: "xw90rybGcwte4JMCM608fCwUMPtnHaVt6kpLMAgml9osFOR7BAdUi8XzGrxQTD3yT_MSq11OL8gXG2fav3pHkQ==",
		},
		WeatherAPIConfig: OpenWeatherMapConfig{
			URL:       "http://api.openweathermap.org",
			Token:     "4805d72f92a507f6872ebcc184915143",
			Latitude:  -19.2569391,
			Longitude: 146.8239537,
		},
		RemoteUnitConfigs: []RemoteUnitConfig{
			{
				UnitName:   "unit_1",
				UnitNumber: 1,
			},
			// {
			// 	UnitName:   "unit_2",
			// 	BLEAddress: "",
			// 	UnitNumber: 2,
			// },
			// {
			// 	UnitName:   "unit_3",
			// 	BLEAddress: "FF:FF:FF:FF:FF:FF",
			// 	UnitNumber: 3,
			// },
			// {
			// 	UnitName:   "unit_4",
			// 	BLEAddress: "FF:FF:FF:FF:FF:FF",
			// 	UnitNumber: 4,
			// },
			// {
			// 	UnitName:   "unit_5",
			// 	BLEAddress: "",
			// 	UnitNumber: 5,
			// },
		},
	}
}
