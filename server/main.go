package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go-upload-chunk/server/config"
	"go-upload-chunk/server/drivers/logger"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	logger.SetupLogger()

	app := gin.Default()
	switch config.Mode() {
	case "prod":
		gin.SetMode(gin.ReleaseMode)
	default:
		gin.SetMode(gin.DebugMode)
	}

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port()),
		Handler: app,
	}

	chanSignal := make(chan os.Signal, 1)
	chanErr := make(chan error, 1)
	chanQuit := make(chan struct{}, 1)

	signal.Notify(chanSignal, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// spawn goroutine : standby waiting signal interrupt or signal error
	go func() {
		for {
			select {
			case <-chanSignal:
				logrus.Warn("receive interrupt signal")
				chanQuit <- struct{}{}
				return
			case e := <-chanErr:
				logrus.Errorf("receive error signal : %s", e.Error())
				chanQuit <- struct{}{}
				return
			}
		}
	}()

	// spawn goroutine : runs http server
	go func() {
		logrus.Infof("Start HTTP Server Listening on Port %d", config.Port())
		if err := httpServer.ListenAndServe(); err != nil {
			chanErr <- err
			return
		}
	}()

	// wait chanQuit
	_ = <-chanQuit

	// close all channels
	close(chanQuit)
	close(chanErr)
	close(chanSignal)

	logrus.Infof("Server Has Exited")
}
