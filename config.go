package main

type Config struct {
	Token          string
	IP             string
	Port           string
	Loglevel       string `default:"info"`
	LogFile        string `default:"/tmp/thermia-tibber.log"`
	CheapStartTemp int
	CheapStopTemp  int
}

func NewConfig() *Config {
	return &Config{}
}
