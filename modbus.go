package main

import (
	"bytes"
	"encoding/binary"

	"github.com/goburrow/modbus"
)

func decode(data []byte) int {
	var i int16
	buf := bytes.NewBuffer(data)
	binary.Read(buf, binary.BigEndian, &i)
	return int(i)
}

func ReadInputRegister(m modbus.Client, address uint16) (int, error) {
	b, err := m.ReadInputRegisters(address, 1)
	return decode(b), err
}

func ReadHoldingRegister(m modbus.Client, address uint16) (int, error) {
	b, err := m.ReadHoldingRegisters(address, 1)
	return decode(b), err
}
