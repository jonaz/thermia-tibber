package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"os"

	"github.com/goburrow/modbus"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetOutput(os.Stdout)
	var write = flag.Bool("write", false, "write")
	var address = flag.Int("address", 0, "address")
	var value = flag.Int("value", 0, "address")
	flag.Parse()
	client := modbus.TCPClient("192.168.10.100:502")

	if *write {
		_, err := client.WriteSingleRegister(uint16(*address-1), uint16(*value))
		if err != nil {
			logrus.Error(err)
			return
		}
		return
	}

	// var i uint16
	// for i < 250 {
	// debugValue(client, "", i)
	// i++
	//}

	debugValue(client, "Start temperature tap water", 22)
	debugValue(client, "Stop temperature tap water", 23)
	debugValue(client, "", 24)
	debugValue(client, "Tap water weighted temperature", 17)
	// Read input register 9
}
func decode(data []byte) float64 {
	var i int16
	buf := bytes.NewBuffer(data)
	binary.Read(buf, binary.BigEndian, &i)
	return float64(i)
}

func ReadHoldingRegister(m modbus.Client, address uint16) (float64, error) {
	b, err := m.ReadHoldingRegisters(address, 1)
	return decode(b), err
}

func ReadInputRegister(m modbus.Client, address uint16) (float64, error) {
	b, err := m.ReadInputRegisters(address, 1)
	return decode(b), err
}

func debugValue(client modbus.Client, text string, address uint16) {
	f, err := ReadInputRegister(client, address)
	if err != nil {
		logrus.Error(err)
		return
	}
	logrus.Infof("%s (%d): %f", text, address, f)

	f, err = ReadHoldingRegister(client, address)
	if err != nil {
		logrus.Error(err)
		return
	}
	logrus.Infof("holding: %s (%d): %f", text, address, f)
}

/*

detta sätter Start till 44 grader:
go run . -write -address 23 -value 4400

detta sätter STOP till 54 grader:
$ go run . -write -address 24 -value 5400


INFO[0000] Start temperature tap water (22): 500.000000
INFO[0001] holding: Start temperature tap water (22): 4300.000000
INFO[0001] Stop temperature tap water (23): 0.000000
INFO[0002] holding: Stop temperature tap water (23): 5400.000000
INFO[0002]  (24): -500.000000
INFO[0003] holding:  (24): 0.000000
INFO[0003] Tap water weighted temperature (17): 5660.000000
INFO[0004] holding: Tap water weighted temperature (17): 0.000000

*/
