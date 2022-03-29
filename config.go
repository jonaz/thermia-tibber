package main

type Config struct {
	Token string `json:"token"`
}

func NewConfig() *Config {
	return &Config{}
}
