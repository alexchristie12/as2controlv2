package db

import (
	"as2controlv2/config"
	"as2controlv2/weather"
	"context"
	"errors"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	influxAPI "github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

// DB Connection to InfluxDB
type DBConnection struct {
	client   influxdb2.Client
	writeAPI influxAPI.WriteAPIBlocking
}

// Tags for a given metric
type Tags struct {
	SystemName     string
	RemoteUnitName string
}

// The current local values for a remote unit. Essentially the average between the sensors
type CurrentLocalValues struct {
	Temperature  float64
	Humidity     float64
	SoilMoisture float64
	FlowRate     float64
	WaterOn      float64
}

// Establish a connection to InfluxDB
func DBInit(conf config.InfluxDBConfig) (DBConnection, error) {
	client := influxdb2.NewClient(conf.URL, conf.Token)

	// Open a write API
	writeAPI := client.WriteAPIBlocking(conf.Organisation, conf.Bucket)
	conn := DBConnection{
		client:   client,
		writeAPI: writeAPI,
	}
	return conn, nil
}

// This method is intended to write a sensor metric to InfluxDB
func (db *DBConnection) WriteSensorMetric(measurementName string, value float64, tags Tags) error {
	// Make the tags
	tagsMap := map[string]string{
		"system_name":      tags.SystemName,
		"remote_unit_name": tags.RemoteUnitName,
	}

	fieldsMap := map[string]interface{}{
		"value": value,
	}

	// Make the fields
	point := write.NewPoint(measurementName, tagsMap, fieldsMap, time.Now())
	if err := db.writeAPI.WritePoint(context.Background(), point); err != nil {
		return err
	}
	return nil
}

// This method is intended to tell the user that a remote unit is not connected
func (db *DBConnection) WriteStatusMetric(tags Tags, status uint) error {
	tagsMap := map[string]string{
		"system_name":      tags.SystemName,
		"remote_unit_name": tags.RemoteUnitName,
	}
	fieldsMap := map[string]interface{}{
		"status": status,
	}

	// Write these values
	point := write.NewPoint("remote_unit_status", tagsMap, fieldsMap, time.Now())
	if err := db.writeAPI.WritePoint(context.Background(), point); err != nil {
		return err
	}
	return nil
}

// Write a current weather prediction from Openweather map
func (db *DBConnection) WriteCurrentWeatherData( /*Need to implement*/ ) error {
	return errors.New("not yet implemented")
}

// Write the metrics for a single unit
func (db *DBConnection) WriteUnitMetrics(measurementName string, localValues CurrentLocalValues, tags Tags) error {
	tagsMap := map[string]string{
		"system_name":      tags.SystemName,
		"remote_unit_name": tags.RemoteUnitName,
	}
	fieldsMap := map[string]interface{}{
		"temperature":   localValues.Temperature,
		"humidity":      localValues.Humidity,
		"soil_moisture": localValues.SoilMoisture,
		"flow_rate":     localValues.FlowRate,
		"water_on":      localValues.WaterOn,
	}
	point := write.NewPoint(measurementName, tagsMap, fieldsMap, time.Now())
	if err := db.writeAPI.WritePoint(context.Background(), point); err != nil {
		return err
	}
	return nil
}

// Write the weather metrics to InfluxDB
func (db *DBConnection) WriteWeatherMetrics(wr weather.CurrentWeatherResult) error {
	tags := map[string]string{
		"location": wr.Name,
	}

	fields := map[string]interface{}{
		"temperature":            wr.Main.TempKelvin - 272.15,
		"temperature_feels_like": wr.Main.TempFeelsLikeKelvin - 272.15,
		"temperature_max":        wr.Main.TempMaxKelvin - 272.15,
		"temperature_min":        wr.Main.TempMinKelvin - 272.15,
		"pressure":               wr.Main.PressurehPa,
		"humidity":               wr.Main.HumidityPercent,
		"cloud_coverage":         wr.Clouds.All,
		"wind_speed_ms":          wr.Wind.Speed,
	}
	point := write.NewPoint("weather", tags, fields, time.Now())
	if err := db.writeAPI.WritePoint(context.Background(), point); err != nil {
		return err
	}
	return nil
}
