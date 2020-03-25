package main

import (
	"flag"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	logLevel := flag.String("loglevel", "debug", "debug, info, warning, error")
	cacheTimeout0 := flag.Int("cache-timeout", -1, "Cache lifespan in milliseconds. -1 disables cache. defaults to -1")
	flag.Parse()

	gin.SetMode(gin.ReleaseMode)
	switch *logLevel {
	case "debug":
		gin.SetMode(gin.DebugMode)
		logrus.SetLevel(logrus.DebugLevel)
		break
	case "warning":
		logrus.SetLevel(logrus.WarnLevel)
		break
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
		break
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}

	logrus.Infof("Starting Docker Info...")

	h, err := NewHTTPServer(*cacheTimeout0)
	if err != nil {
		logrus.Errorf("Error preparing HTTPServer. err=%s", err)
		os.Exit(1)
	}

	err = h.Start()
	if err != nil {
		logrus.Errorf("Error starting server. err=%s", err)
		os.Exit(1)
	}

}
