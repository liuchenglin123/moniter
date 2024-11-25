package conf

import (
	"encoding/json"
	"os"
)

type ServerConfig struct {
	IP           string   `json:"ip"`
	ProcessNames []string `json:"process_names"`
	IntervalTime int      `json:"interval_time"`
	DB           struct {
		User     string `json:"user"`
		Password string `json:"password"`
		Host     string `json:"host"`
		Port     string `json:"port"`
		Database string `json:"database"`
	} `json:"db"`
}

var Sc *ServerConfig

func Init() {
	file, err := os.ReadFile("./config.json")
	if err != nil {
		panic(err)
	}
	Sc = &ServerConfig{}
	err = json.Unmarshal(file, Sc)
	if err != nil {
		panic(err)
	}
}
