package serial

import (
	"as2controlv2/config"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tarmSerial "github.com/tarm/serial"
)

type SerialConnection struct {
	conn           *tarmSerial.Port
	receieveBuffer [512]byte
	CurrentDevice  uint // The current zone that the bluetooth module is connected to
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

func (sc *SerialConnection) sendPollCmd(deviceNumber uint) (string, error) {
	// // First write to the serial connection, to clear the buffer
	_, err := sc.conn.Write([]byte(" \n"))
	if err != nil {
		return "", err
	}
	pollStr := fmt.Sprintf("poll=%d\n", deviceNumber)
	bytesWrote, err := sc.conn.Write([]byte(pollStr))
	if bytesWrote != len(pollStr) {
		// This means that we failed to write the serial connection
		return "", err
	}
	if err != nil {
		return "", err
	}
	time.Sleep(100 * time.Millisecond)

	// In this case we don't care about how many bytes were read, as this is handling in the parsing
	// We need to read until we get a \n
	readContents := make([]byte, 0)
	// This needs a timeout, lets set it at 5 seconds
	timeout := time.Now().Add(5 * time.Second)
	for {
		if time.Now().After(timeout) {
			return "", errors.New("timed out of receive loop")
		}
		n, err := sc.conn.Read(sc.receieveBuffer[:])
		if err != nil {
			return "", err
		}
		readContents = append(readContents, sc.receieveBuffer[:n]...)
		if strings.Contains(string(sc.receieveBuffer[:n]), "\n") {
			break
		}
	}
	// Otherwise parse out everything. It is all in keyvalue pairs
	byteStr := string(readContents)
	return byteStr, nil
}

// Poll a device for its sensor information, output will in as key=value,... pairs
// Will return an error if it fails, or timesout
func (sc *SerialConnection) PollDevice(deviceNumber uint) ([]SensorReading, error) {
	var byteStr string
	byteStr, err := sc.sendPollCmd(deviceNumber)
	if err != nil {
		return nil, err
	}
	fmt.Println("Read contents")
	fmt.Println(byteStr)
	// Retry 5 times
	for i := 0; i < 5; i++ {
		if strings.Contains(byteStr, "CMD") {
			// We are still in command mode, try again
			sc.CurrentDevice = 0
			log.Println("still in command mode, retrying")
			sc.WriteToDevice("---\n\r")
			if err := sc.SwitchDevice(deviceNumber); err != nil {
				return nil, err
			}
			// And now retry
			byteStr, err = sc.sendPollCmd(deviceNumber)
			if err != nil {
				return nil, err
			}
		} else {
			break
		}
	}
	// This means it is still broken
	if strings.Contains(byteStr, "CMD") {
		return nil, errors.New("failed to change device")
	}
	// Split on commas
	sensorParts := strings.Split(strings.Trim(byteStr, "\r\n\t "), ",")
	sensorReadings := make([]SensorReading, len(sensorParts))
	// Now separate out on the '='
	for i, sp := range sensorParts {
		readingParts := strings.Split(sp, "=")
		if len(readingParts) < 2 {
			continue
		}
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

func (sc *SerialConnection) SwitchDevice(newDevice uint) error {
	if newDevice == sc.CurrentDevice {
		return nil // We are already on the correct device
	}
	if err := sc.WriteToDevice("$$$"); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)

	if err := sc.WriteToDevice("k,1\n\r"); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)

	if err := sc.WriteToDevice(fmt.Sprintf("c%d\n\r", newDevice)); err != nil {
		return err
	}
	time.Sleep(2000 * time.Millisecond)
	if err := sc.WriteToDevice(" \r\n"); err != nil {
		return err
	}
	sc.CurrentDevice = newDevice
	log.Println("switched to device: ", sc.CurrentDevice)
	sc.conn.Flush()
	return nil
}

// Polls a device for a given amount of time, returns whatever its gets after that interval
func (sc *SerialConnection) PollDeviceForDuration(timeout time.Duration) (string, error) {
	readContents := make([]byte, 0)
	// This needs a timeout, lets set it at 5 seconds
	timeoutTime := time.Now().Add(timeout)
	for {
		if time.Now().After(timeoutTime) {
			break
		}
		n, err := sc.conn.Read(sc.receieveBuffer[:])
		if err != nil {
			return "", err
		}
		readContents = append(readContents, sc.receieveBuffer[:n]...)
	}
	// Otherwise parse out everything. It is all in keyvalue pairs
	byteStr := string(readContents)
	return byteStr, nil
}
