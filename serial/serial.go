package serial

import (
	"as2controlv2/config"
	"fmt"
	"strconv"
	"strings"
	"time"

	tarmSerial "github.com/tarm/serial"
)

type SerialConnection struct {
	conn           *tarmSerial.Port
	receieveBuffer [16384]byte
}

type SensorReading struct {
	Name  string
	Value float32
}

func SerialConnectionInit(conf config.SerialConfig) (SerialConnection, error) {
	c := tarmSerial.Config{
		Name:        conf.Port,
		Baud:        int(conf.BaudRate),
		ReadTimeout: time.Second * time.Duration(conf.TimeoutSeconds),
	}
	s, err := tarmSerial.OpenPort(&c)
	if err != nil {
		return SerialConnection{}, err
	}
	serialConn := SerialConnection{
		conn: s,
	}
	return serialConn, nil
}

// Poll a device for its sensor information, output will in as key=value,... pairs
// Will return an error if it fails, or timesout
func (sc *SerialConnection) PollDevice(deviceNumber uint) ([]SensorReading, error) {
	// First write to the serial connection
	pollStr := fmt.Sprintf("poll=%d\n", deviceNumber)
	bytesWrote, err := sc.conn.Write([]byte(fmt.Sprintf(pollStr)))
	if bytesWrote != len(pollStr) {
		// This means that we failed to write the serial connection
		return nil, err
	}

	// In this case we don't care about how many bytes were read, as this is handling in the parsing
	_, err = sc.conn.Read(sc.receieveBuffer[:])
	if err != nil {
		return nil, err
	}
	// Otherwise parse out everything. It is all in keyvalue pairs
	byteStr := string(sc.receieveBuffer[:])
	// Split on commas
	sensorParts := strings.Split(byteStr, ",")
	sensorReadings := make([]SensorReading, len(sensorParts))
	// Now separate out on the '='
	for i, sp := range sensorParts {
		readingParts := strings.Split(sp, "=")
		// First part is name, second is reading, so parse the reading
		value, err := strconv.ParseFloat(readingParts[1], 32)
		if err != nil {
			return nil, err
		}
		sensorReadings[i] = SensorReading{Name: readingParts[0], Value: float32(value)}
	}
	return sensorReadings, nil
}
