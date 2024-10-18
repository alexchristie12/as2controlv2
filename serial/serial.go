package serial

import (
	"as2controlv2/config"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	tarmSerial "github.com/tarm/serial"
)

type SerialConnection struct {
	conn           *tarmSerial.Port
	receieveBuffer [512]byte
}

type SensorReading struct {
	Name  string
	Value float64
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
	bytesWrote, err := sc.conn.Write([]byte(pollStr))
	if bytesWrote != len(pollStr) {
		// This means that we failed to write the serial connection
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	time.Sleep(500 * time.Millisecond)

	// In this case we don't care about how many bytes were read, as this is handling in the parsing
	// We need to read until we get a \n
	readContents := make([]byte, 0)
	for {
		n, err := sc.conn.Read(sc.receieveBuffer[:])
		if err != nil {
			return nil, err
		}
		readContents = append(readContents, sc.receieveBuffer[:n]...)
		if strings.Contains(string(sc.receieveBuffer[:n]), "\n") {
			break
		}
	}
	// Otherwise parse out everything. It is all in keyvalue pairs
	byteStr := string(readContents)
	// fmt.Println("Read contents")
	// fmt.Println(byteStr)
	// Split on commas
	sensorParts := strings.Split(strings.Trim(byteStr, "\r\n\t "), ",")
	sensorReadings := make([]SensorReading, len(sensorParts))
	// Now separate out on the '='
	for i, sp := range sensorParts {
		readingParts := strings.Split(sp, "=")
		// First part is name, second is reading, so parse the reading
		readingParts[0] = strings.Trim(readingParts[0], "\r\n\t ")
		readingParts[1] = strings.Trim(readingParts[1], "\r\n\t ")
		value, err := strconv.ParseFloat(readingParts[1], 64)
		if err != nil {
			return nil, err
		}
		sensorReadings[i] = SensorReading{Name: readingParts[0], Value: value}
	}
	return sensorReadings, nil
}

func (sc *SerialConnection) WriteToDevice(msg string) error {
	n, err := sc.conn.Write([]byte(msg))
	if n != len(msg) {
		return errors.New("could not write full length of message")
	}
	if err != nil {
		return err
	}
	return nil
}
