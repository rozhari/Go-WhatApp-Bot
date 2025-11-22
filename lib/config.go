package lib

import (
	"time"

	"go.mau.fi/whatsmeow"
	"github.com/joho/godotenv"
)

type Configuration struct {
	HANDLERS  string
	SUDO      string
	MODE      string
	LOG_MSG   bool
	READ_MSG  bool
	READ_CMD  bool
	ERROR_MSG bool
}

var Config Configuration
var Client *whatsmeow.Client
var StartTime time.Time

func LoadConfig() {
	_ = godotenv.Load()

	Config = Configuration{
		HANDLERS:  getEnv("HANDLERS", "."),
		SUDO:      getEnv("SUDO", ""),
		MODE:      getEnv("MODE", "public"),
		LOG_MSG:   getEnvBool("LOG_MSG", true),
		READ_MSG:  getEnvBool("READ_MSG", true),
		READ_CMD:  getEnvBool("READ_CMD", true),
		ERROR_MSG: getEnvBool("ERROR_MSG", true),
	}
}