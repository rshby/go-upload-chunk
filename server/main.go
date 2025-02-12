package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
	"go-upload-chunk/server/config"
	"go-upload-chunk/server/drivers/logger"
	"go-upload-chunk/server/http/router"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
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

	// init validator
	validate := validator.New()

	// Setup Router
	router.SetupRouter(&app.RouterGroup, validate)

	// create http server
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
				logrus.Warn("receive interrupt signal ⚠️")
				gracefullShutdown(httpServer)
				chanQuit <- struct{}{}
				return
			case e := <-chanErr:
				logrus.Errorf("receive error signal : %s", e.Error())
				gracefullShutdown(httpServer)
				chanQuit <- struct{}{}
				return
			}
		}
	}()

	// spawn goroutine : runs http server
	go func() {
		logrus.Infof("Start HTTP Server Listening on Port %d ⏳", config.Port())
		if err := httpServer.ListenAndServe(); err != nil {
			chanErr <- err
			return
		}
	}()

	// waiting interrupt
	_ = <-chanQuit

	// close all channels
	close(chanQuit)
	close(chanErr)
	close(chanSignal)

	logrus.Infof("Server Has Exited 🛑")
}

func gracefullShutdown(httpServer *http.Server) {
	if httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			_ = httpServer.Close()
			logrus.Warn("force close HTTP Server ⚠️")
			return
		}

		_ = httpServer.Close()
		logrus.Infof("gracefull shutdown HTTP Server ❎")
	}
}
