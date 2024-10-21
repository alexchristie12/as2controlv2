package db

import (
	"as2controlv2/config"
	"as2controlv2/serial"
	"context"
	"errors"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	influxAPI "github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

type DBConnection struct {
	client   influxdb2.Client
	writeAPI influxAPI.WriteAPIBlocking
}

type Tags struct {
	SystemName     string
	RemoteUnitName string
}

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

// This function write multiple sensor readings pulling the the correspoding remote unit, hence the config
func (db *DBConnection) WriteSensorReadings(readings []serial.SensorReading, conf config.RemoteUnitConfig, systemName string) error {
	// Construct the tags
	tags := Tags{
		SystemName:     systemName,
		RemoteUnitName: conf.UnitName,
	}
	for _, r := range readings {
		// Should do this after it is averaged out
		err := db.WriteSensorMetric(r.Name, float64(r.Value), tags)
		if err != nil {
			return err
		}
	}
	return nil
}

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
