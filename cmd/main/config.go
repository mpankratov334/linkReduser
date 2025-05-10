package main

import (
	"flag"
	"os"
)

type Config struct {
	serverAddress string
	saveAddress   string
	dbHost        string
	storeFile     string
}

var adr = "localhost:8080"
var saveAdr = "localhost:8080"
var dbHost = "localhost"
var storeFilePath = ""

func setConfig() Config {
	adr = *flag.String("a", adr, "server address")
	saveAdr = *flag.String("b", saveAdr, "saving address")
	dbHost = *flag.String("d", dbHost, "host of database")
	storeFilePath = *flag.String("f", storeFilePath, "host of database")
	flag.Parse()

	if envServerAddress := os.Getenv("SERVER_ADDRESS"); envServerAddress != "" {
		adr = envServerAddress
	}
	if envSaveAddress := os.Getenv("BASE_URL"); envSaveAddress != "" {
		saveAdr = envSaveAddress
	}
	if envDBHost := os.Getenv("DATABASE_DSN"); envDBHost != "" {
		adr = envDBHost
	}

	if envStoreFile := os.Getenv("FILE_STORAGE_PATH"); envStoreFile != "" {
		storeFilePath = envStoreFile
	}

	config := Config{
		serverAddress: adr,
		saveAddress:   saveAdr,
		dbHost:        dbHost,
		storeFile:     storeFilePath,
	}
	return config
}
