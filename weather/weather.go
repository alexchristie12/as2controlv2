package weather

import (
	"as2controlv2/config"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// This package is intended to pull data from Openweather map, for now we are only doing current data
// not predicitons yet

type WeatherAPI struct {
	URL       string
	Token     string
	Latitude  float64
	Longitude float64
}

type CurrentWeatherResult struct {
	Coord      Coord   `json:"coord"`
	Weather    Weather `json:"weather"`
	Base       string  `json:"base"`
	Main       Main    `json:"main"`
	Visibility uint    `json:"visibility"`
	Wind       Wind    `json:"wind"`
	Clouds     Clouds  `json:"clouds"`
	DT         uint    `json:"dt"`
	Sys        Sys     `json:"sys"`
	Timezone   uint    `json:"timezone"`
	ID         uint    `json:"id"`
	Name       string  `json:"name"`
	Cod        int     `json:"cod"`
}

type Weather struct {
	ID          int    `json:"id"`
	Main        string `json:"main"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

type Coord struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"long"`
}

type Main struct {
	TempKelvin          float64 `json:"temp"`
	TempFeelsLikeKelvin float64 `json:"feels_like"`
	TempMinKelvin       float64 `json:"temp_min"`
	TempMaxKelvin       float64 `json:"temp_max"`
	PressurehPa         float64 `json:"pressure"`
	HumidityPercent     float64 `json:"humidity"`
	SeaLevelPressure    float64 `json:"sea_level"`
	GroundLevelPressure float64 `json:"grnd_level"`
}

type Wind struct {
	Speed float64 `json:"speed"`
	Deg   float64 `json:"deg"`
}

type Clouds struct {
	All float64 `json:"all"`
}

type Sys struct {
	Type    int    `json:"type"`
	ID      int    `json:"id"`
	Country string `json:"country"`
	Sunrise uint   `json:"sunrise"`
	Sunset  uint   `json:"sunset"`
}

func WeatherInit(conf config.OpenWeatherMapConfig) WeatherAPI {
	return WeatherAPI{
		URL:       conf.URL,
		Token:     conf.Token,
		Latitude:  conf.Latitude,
		Longitude: conf.Longitude,
	}
}

func (w *WeatherAPI) GetCurrentWeather() (CurrentWeatherResult, error) {
	fullUrl := fmt.Sprintf("%s/data/2.5/weather?lat=%f&long=%f&appid=%s", w.URL, w.Latitude, w.Longitude, w.Token)
	resp, err := http.Get(fullUrl)
	if err != nil {
		return CurrentWeatherResult{}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return CurrentWeatherResult{}, errors.New(fmt.Sprint("unexpected status code: ", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return CurrentWeatherResult{}, err
	}

	var result CurrentWeatherResult
	err = json.Unmarshal(body, &result)
	if err != nil {
		return CurrentWeatherResult{}, err
	}

	return result, nil
}
