package config

import "encoding/json"

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
	BLEAddress string `json:"ble_address"`
	UnitNumber uint   `json:"number"`
	SoilType   string `json:"soil_type"` // Can be clay, sand, loam
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

func MakeExampleConfig() Config {
	return Config{
		Name:                   "example_system_config",
		WeatherIntervalSeconds: 3600,
		RemoteIntervalSeconds:  60,
		SerialConfig: SerialConfig{
			Port:           "/dev/tty2s",
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
				BLEAddress: "FF:FF:FF:FF:FF:FF",
				UnitNumber: 1,
			},
			{
				UnitName:   "unit_2",
				BLEAddress: "FF:FF:FF:FF:FF:FF",
				UnitNumber: 2,
			},
			{
				UnitName:   "unit_3",
				BLEAddress: "FF:FF:FF:FF:FF:FF",
				UnitNumber: 3,
			},
			{
				UnitName:   "unit_4",
				BLEAddress: "FF:FF:FF:FF:FF:FF",
				UnitNumber: 4,
			},
			{
				UnitName:   "unit_5",
				BLEAddress: "FF:FF:FF:FF:FF:FF",
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
