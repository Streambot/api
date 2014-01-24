package main

import (
	"streambot"
	"io/ioutil"
	"encoding/json"
	"os/signal"
	stdlog "log"
    "os"
    "github.com/op/go-logging"
)

var log = logging.MustGetLogger("streambot-api")

type ServerConfig struct {
	Port int `json:"port"`
}

type StatsConfig struct {
	Port int `json:"port"`
}

type DatabaseConfig struct {
	Port  uint16 `json:"port"`
	Host  string `json:"host"`
	Graph string `json:"graph"`
}

type Config struct {
	Server 	 ServerConfig 	`json:"server"`
	Database DatabaseConfig `json:"database"`
	Stats	 StatsConfig	`json:"stats"`
	Debug	 bool			`json:"debug"`
}

func NewConfigurationFromJSONFile(file string) (err error, config Config) {
	buf, err := ioutil.ReadFile(file)
    if err != nil {
        return
    }
    json.Unmarshal(buf, &config)
    return
}

func ReadConfig() Config {
	file := "./config.json"
	err, config := NewConfigurationFromJSONFile(file)
	if err != nil {
		log.Fatal("Unexpected error on loading configuration from JSON file `%s`: %v", file, err)
	}
	return config
}

var config = ReadConfig()

func init() {
	// Customize the output format
    logging.SetFormatter(logging.MustStringFormatter("%{message}"))
    // Setup one stdout and one syslog backend.
    logBackend := logging.NewLogBackend(os.Stderr, "", stdlog.LstdFlags/*|stdlog.Lshortfile*/)
    logBackend.Color = true
    syslogBackend, err := logging.NewSyslogBackend("")
    if err != nil {
        log.Fatal(err)
    }
    // Combine them both into one logging backend.
    logging.SetBackend(logBackend, syslogBackend)
    if config.Debug == true {
    	logging.SetLevel(logging.DEBUG, "streambot")
    }
}

func main() {
	db := streambot.NewGraphDatabase(
		config.Database.Graph, 
		config.Database.Host, 
		config.Database.Port)
	api := streambot.NewAPI(db)
	errChan := make(chan error, 1)
	basePath := "/v1/"
	log.Info("Running API server on Port %d at base path %s", config.Server.Port, basePath)
	api.Serve(config.Server.Port, basePath, errChan)  
	go func() {
		err := <- errChan
		log.Error("Unexpected error occurred when starting API: %v", err)
	}()
	c := make(chan os.Signal, 1)                                       
	signal.Notify(c, os.Interrupt)                                     
	go func() {                                                        
	  for sig := range c {                                             
	    log.Debug("Captured %v, stopping API server..", sig)
	    api.Shutdown()                                                
	  }                                                                
	}()
	<- api.Closed
	close(errChan)
	err := <- errChan
	if err != nil {
		log.Error("Unexpected error occurred when starting API: %v", err)
	}
	log.Info("Finish.") 
	os.Exit(0)   
}