package main

type Config struct {
	Token    string
	IP       string
	Port     string
	Loglevel string `default:"info"`
}

func NewConfig() *Config {
	return &Config{}
}
