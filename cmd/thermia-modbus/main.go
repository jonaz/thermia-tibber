package main

import (
	"bytes"
	"encoding/binary"
	"flag"

	"github.com/goburrow/modbus"
	"github.com/sirupsen/logrus"
)

func main() {
	var write = flag.Bool("write", false, "write")
	var address = flag.Int("address", 0, "address")
	var value = flag.Int("value", 0, "address")
	client := modbus.TCPClient("192.168.10.100:502")

	if *write {
		_, err := client.WriteSingleRegister(uint16(*address-1), uint16(*value))
		if err != nil {
			logrus.Error(err)
			return
		}
		return
	}

	debugValue(client, "tart temperature tap water", 22)
	debugValue(client, "Stop temperature tap water", 23)
	debugValue(client, "Tap water weighted temperature", 17)
	// Read input register 9
}
func decode(data []byte) float64 {
	var i int16
	buf := bytes.NewBuffer(data)
	binary.Read(buf, binary.BigEndian, &i)
	return float64(i)
}

func ReadInputRegister(m modbus.Client, address uint16) (float64, error) {
	b, err := m.ReadInputRegisters(address-1, 1)
	return decode(b), err
}

func debugValue(client modbus.Client, text string, address uint16) {
	f, err := ReadInputRegister(client, address)
	if err != nil {
		logrus.Error(err)
		return
	}
	logrus.Infof("%s: %f", text, f)
}
