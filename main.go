package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goburrow/modbus"
	"github.com/koding/multiconfig"
	"github.com/sirupsen/logrus"
)

var httpClient = &http.Client{
	Timeout: time.Second * 30,
}

var pricesStore = NewPrices()

var wg = &sync.WaitGroup{}
var fileLogger = logrus.New()

// example run: -token asdf -ip 192.168.10.100 -port 502 -loglevel debug.
func main() {
	config := NewConfig()
	multiconfig.MustLoad(config)

	lvl, err := logrus.ParseLevel(config.Loglevel)
	if err != nil {
		log.Println("error setting logrus loglevel: ", err)
		return
	}
	logrus.SetLevel(lvl)

	file, err := os.OpenFile(config.LogFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)
	if err != nil {
		logrus.Fatal(err)
	}
	defer file.Close()
	fileLogger.SetOutput(file)

	r := gin.Default()
	r.StaticFile("/", config.LogFile)
	go func() {
		err := r.Run(":9191")
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logrus.Error(err)
		}
	}()

	client := modbus.TCPClient(net.JoinHostPort(config.IP, config.Port))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	wg.Add(2)
	defer stop()
	go tickerPriceFetchLoop(ctx, config)
	go tickerSyncHeapPump(ctx, client, config)
	wg.Wait()
}

func tickerPriceFetchLoop(ctx context.Context, config *Config) {
	defer wg.Done()
	delay := nextDelay()
	timer := time.NewTimer(delay)
	fetchAndCalculate(config)
	logrus.Info("scheduling first run in", delay)
	for {
		select {
		case <-timer.C:
			timer.Reset(nextDelay())
			fetchAndCalculate(config)
		case <-ctx.Done():
			return
		}
	}
}

func tickerSyncHeapPump(ctx context.Context, client modbus.Client, config *Config) {
	defer wg.Done()
	timer := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-timer.C:
			err := syncHeatPump(client, config)
			if err != nil {
				logrus.Error(err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func syncHeatPump(client modbus.Client, config *Config) error {
	start, stop, tank, err := getTemps(client)
	if err != nil {
		return err
	}

	if isSameHourAndDay(time.Now(), pricesStore.cheapestHour) {
		if stop == config.CheapStopTemp && start == config.CheapStartTemp {
			fileLogger.
				WithField("tank", tank).
				Info("temps already updated")
			return nil
		}

		err := writeTemps(client, config.CheapStartTemp, config.CheapStopTemp)
		if err != nil {
			return err
		}

		fileLogger.
			WithField("start", config.CheapStartTemp).
			WithField("stop", config.CheapStopTemp).
			WithField("tank", tank).
			Info("updating temperatures")

		return nil
	}

	if stop == config.NormalStopTemp && start == config.NormalStartTemp {
		fileLogger.
			WithField("tank", tank).
			Info("temps already updated")
		return nil
	}

	// set back default values
	err = writeTemps(client, config.NormalStartTemp, config.NormalStopTemp)
	if err != nil {
		return err
	}

	fileLogger.
		WithField("start", config.NormalStartTemp).
		WithField("stop", config.NormalStopTemp).
		WithField("tank", tank).
		Info("updating temperatures")
	return nil
}

func getTemps(client modbus.Client) (start, stop int, tank float64, err error) {
	start, err = ReadHoldingRegister(client, 22)
	if err != nil {
		return
	}
	start /= 100

	stop, err = ReadHoldingRegister(client, 23)
	if err != nil {
		return
	}
	stop /= 100

	tankInt, err := ReadInputRegister(client, 17)
	if err != nil {
		return
	}
	tank = float64(tankInt) / 100.0
	return
}

func writeTemps(client modbus.Client, start, stop int) error {
	_, err := client.WriteSingleRegister(22, uint16(start*100)) // 100 = 1c
	if err != nil {
		return fmt.Errorf("error writeTemps 22: %w", err)
	}

	_, err = client.WriteSingleRegister(23, uint16(stop*100))
	if err != nil {
		return fmt.Errorf("error writeTemps 23: %w", err)
	}
	return nil
}
