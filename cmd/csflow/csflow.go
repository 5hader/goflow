package main

import (
	"flag"
	"fmt"
	"github.com/cloudflare/goflow/transport"
	"github.com/cloudflare/goflow/utils"
	log "github.com/sirupsen/logrus"
	"os"
	"runtime"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

var (
	version    = ""
	AppVersion = "GoFlow sFlow " + version

	Addr = flag.String("addr", "", "sFlow listening address")
	Port = flag.Int("port", 6343, "sFlow listening port")

	Workers  = flag.Int("workers", 1, "Number of sFlow workers")
	LogLevel = flag.String("loglevel", "info", "Log level")
	LogFmt   = flag.String("logfmt", "normal", "Log formatter")

	EnableKafka = flag.Bool("kafka", true, "Enable Kafka")
	MetricsAddr = flag.String("metrics.addr", ":8080", "Metrics address")
	MetricsPath = flag.String("metrics.path", "/metrics", "Metrics path")

	Version = flag.Bool("v", false, "Print version")
)

func init() {
	transport.RegisterFlags()
}

func httpServer() {
	http.Handle(*MetricsPath, promhttp.Handler())
	log.Fatal(http.ListenAndServe(*MetricsAddr, nil))
}

func main() {
	flag.Parse()

	if *Version {
		fmt.Println(AppVersion)
		os.Exit(0)
	}

	lvl, _ := log.ParseLevel(*LogLevel)
	log.SetLevel(lvl)
	switch *LogFmt {
	case "json":
		log.SetFormatter(&log.JSONFormatter{})
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	log.Info("Starting GoFlow")

	s := &utils.StateSFlow{
		Transport: &utils.DefaultLogTransport{},
		Logger:    log.StandardLogger(),
	}

	go httpServer()

	if *EnableKafka {
		kafkaState, err := transport.StartKafkaProducerFromArgs(log.StandardLogger())
		if err != nil {
			log.Fatal(err)
		}
		s.Transport = kafkaState
	}
	log.WithFields(log.Fields{
		"Type": "sFlow"}).
		Infof("Listening on UDP %v:%v", *Addr, *Port)

	err := s.FlowRoutine(*Workers, *Addr, *Port)
	if err != nil {
		log.Fatalf("Fatal error: could not listen to UDP (%v)", err)
	}
}
