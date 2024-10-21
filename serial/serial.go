package serial

import (
	"as2controlv2/config"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"strconv"
	"strings"
	"time"

	tarmSerial "github.com/tarm/serial"
)

type SerialConnection struct {
	conn           *tarmSerial.Port
	receieveBuffer [512]byte
	CurrentDevice  uint // The current zone that the bluetooth module is connected to
	logger         *slog.Logger
}

type SensorReading struct {
	Name  string
	Value float64
}

func SerialConnectionInit(conf config.SerialConfig, logger *slog.Logger) (SerialConnection, error) {
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
		conn:   s,
		logger: logger,
	}
	return serialConn, nil
}

func (sc *SerialConnection) sendPollCmd(deviceNumber uint) (string, error) {
	// // First write to the serial connection, to clear the buffer
	_, err := sc.conn.Write([]byte(" \r\n"))
	if err != nil {
		return "", err
	}
	pollStr := fmt.Sprintf("poll=%d\r\n", deviceNumber)
	bytesWrote, err := sc.conn.Write([]byte(pollStr))
	if bytesWrote != len(pollStr) {
		// This means that we failed to write the serial connection
		sc.logger.Error(fmt.Sprintf("Failed to write to unit %d, could not send poll command", deviceNumber))
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
			sc.logger.Warn("timed out of receive loop")
			return "", errors.New("timed out of receive loop")
		}
		n, err := sc.conn.Read(sc.receieveBuffer[:])
		if n == 0 {
			return "EOF", nil // This will never to returned from anything else
		}
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
	byteStr, err := sc.sendPollCmd(deviceNumber) // If it is an EOF error, then that means that we just didn't get anything!

	if byteStr == "EOF" {
		// Retry
		byteStr, err = sc.sendPollCmd(deviceNumber)
	}
	if err != nil && byteStr != "EOF" {
		return nil, err
	}

	// Retry 3 times
	if strings.Contains(byteStr, "CMD") {
		sc.logger.Warn("initial polling failed, retrying")
		for i := 0; i < 3; i++ {
			if strings.Contains(byteStr, "CMD") {
				// We are still in command mode, try again
				sc.CurrentDevice = 0
				sc.logger.Warn("still in command mode, retrying")
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
	}
	// This means it is still broken
	if strings.Contains(byteStr, "CMD") {
		sc.logger.Error("stuck in command mode still")
		return nil, errors.New("failed to change device")
	}
	if byteStr == "" {
		sc.logger.Error("we still couldn't get anything")
		return nil, errors.New("device was unresponsive")
	}
	sc.logger.Info(fmt.Sprintf("got contents: %s", byteStr))
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
	time.Sleep(1000 * time.Millisecond)

	if err := sc.WriteToDevice(fmt.Sprintf("c%d\n\r", newDevice)); err != nil {
		return err
	}
	time.Sleep(3000 * time.Millisecond)
	if err := sc.WriteToDevice(" \r\n"); err != nil {
		return err
	}

	contents, err := sc.PollDeviceForDuration(500 * time.Millisecond)
	if err != nil {
		sc.logger.Error("checking what we get when we switched devices, and encountered an error")
	}
	sc.logger.Info(fmt.Sprintf("connection contents: %s", contents))
	// Read everything that is waiting on the serial line

	// Now if I have %%DISCONNECT%, I need to try again twice
	if strings.Contains(contents, `%%DISCONNECT%`) {
		for i := 0; i > 2; i++ {
			if err := sc.WriteToDevice(fmt.Sprintf("c%d\n\r", newDevice)); err != nil {
				return err
			}
			contents, err = sc.PollDeviceForDuration(500 * time.Millisecond)
			if err != nil {
				return err
			}
			if !strings.Contains(contents, `%%DISCONNECT%`) {
				break
			}
		}
	}
	sc.conn.Flush()
	sc.CurrentDevice = newDevice
	log.Println("switched to device: ", sc.CurrentDevice)
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
		if n == 0 {
			continue
		}
		if err != nil {
			return "", err
		}
		readContents = append(readContents, sc.receieveBuffer[:n]...)
	}
	// Otherwise parse out everything. It is all in keyvalue pairs
	byteStr := string(readContents)
	return byteStr, nil
}
