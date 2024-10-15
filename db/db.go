package db

import (
	"as2controlv2/config"
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
