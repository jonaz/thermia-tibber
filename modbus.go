package main

import (
	"bytes"
	"encoding/binary"

	"github.com/goburrow/modbus"
)

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
