package main

import (
	"streambot"
	"io/ioutil"
	"encoding/json"
	"os/signal"
	stdlog "log"
    "os"
    "github.com/op/go-logging"
    "github.com/jessevdk/go-flags"
    "fmt"
)

type Options struct {
    ConfigFilepath string `short:"c" long:"config" description:"File path of configuration file"`
}

var log = logging.MustGetLogger("streambot-api")

type ServerConfig struct {
	Port int `json:"port"`
}

type StatsConfig struct {
	Port int `json:"port"`
}

type DatabaseConfig struct {
	Hosts []string `json:"hosts"`
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
    err = json.Unmarshal(buf, &config)
    return
}

func ReadConfig(file string) Config {
	err, config := NewConfigurationFromJSONFile(file)
	if err != nil {
		errMsgFormat := "Unexpected error on loading configuration from JSON file `%s`: %v"
		log.Fatal(fmt.Sprintf(errMsgFormat, file, err))
	}
	return config
}

var config Config

func init() {
	var options Options
	var parser = flags.NewParser(&options, flags.Default)
    if _, err := parser.Parse(); err != nil {
    	fmt.Println(fmt.Sprintf("Error when parsing arguments: %v", err))
        os.Exit(1)
    }
    if options.ConfigFilepath == "" {
    	fmt.Println("Missing a valid configuration file specification argument. Usage: -c <config_file>")
    	os.Exit(1)	
    }
    config = ReadConfig(options.ConfigFilepath)
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
    } else {
    	logging.SetLevel(logging.INFO, "streambot")
    }
}

func main() {
	db, err := streambot.NewGraphDatabase(
		config.Database.Graph, 
		config.Database.Hosts,
	)
	if err != nil {
		log.Fatalf("Unexpected error when intializing graph database driver: %v", err)
	}
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
	err = <- errChan
	if err != nil {
		log.Error("Unexpected error occurred when starting API: %v", err)
	}
	log.Info("Finish.") 
	os.Exit(0)   
}