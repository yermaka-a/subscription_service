package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

type Config interface {
	GetPort() int
}

const (
	GRPC_PORT = "GRPC_PORT"
)

type config struct {
	port int
}

func MustLoad() Config {
	port, err := strconv.ParseUint(getValByKey(GRPC_PORT), 10, 64)
	if err != nil {
		panic("port has an incorrect value")
	}
	return &config{
		port: int(port),
	}
}

func getValByKey(key string) string {
	var value string
	flag.StringVar(&value, key, "", "server port")
	flag.Parse()
	if value == "" {
		if value, isExists := os.LookupEnv(key); isExists {
			return value
		}
	} else {
		return value
	}
	panic(fmt.Sprintf("value by key- %s isn't found", key))
}

func (c *config) GetPort() int {
	return c.port
}
