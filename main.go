package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/ccirello/supervisor"
	"github.com/gin-gonic/gin"
	"github.com/iron-io/functions/api/datastore"
	"github.com/iron-io/functions/api/mqs"
	"github.com/iron-io/functions/api/runner"
	"github.com/iron-io/functions/api/runner/task"
	"github.com/iron-io/functions/api/server"
	"github.com/spf13/viper"
)

const (
	envLogLevel = "log_level"
	envMQ       = "mq_url"
	envDB       = "db_url"
	envPort     = "port" // be careful, Gin expects this variable to be "port"
	envAPIURL   = "api_url"
)

func init() {
	cwd, err := os.Getwd()
	if err != nil {
		log.WithError(err).Fatalln("")
	}
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetDefault(envLogLevel, "info")
	viper.SetDefault(envMQ, fmt.Sprintf("bolt://%s/data/worker_mq.db", cwd))
	viper.SetDefault(envDB, fmt.Sprintf("bolt://%s/data/bolt.db?bucket=funcs", cwd))
	viper.SetDefault(envPort, 8080)
	viper.SetDefault(envAPIURL, fmt.Sprintf("http://127.0.0.1:%d", viper.GetInt(envPort)))
	viper.AutomaticEnv() // picks up env vars automatically
	logLevel, err := log.ParseLevel(viper.GetString("log_level"))
	if err != nil {
		log.WithError(err).Fatalln("Invalid log level.")
	}
	log.SetLevel(logLevel)

	gin.SetMode(gin.ReleaseMode)
	if logLevel == log.DebugLevel {
		gin.SetMode(gin.DebugMode)
	}
}

func main() {
	ctx, halt := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		log.Info("Halting...")
		halt()
	}()

	ds, err := datastore.New(viper.GetString(envDB))
	if err != nil {
		log.WithError(err).Fatalln("Invalid DB url.")
	}

	mq, err := mqs.New(viper.GetString(envMQ))
	if err != nil {
		log.WithError(err).Fatal("Error on init MQ")
	}
	metricLogger := runner.NewMetricLogger()

	rnr, err := runner.New(metricLogger)
	if err != nil {
		log.WithError(err).Fatalln("Failed to create a runner")
	}

	svr := &supervisor.Supervisor{
		MaxRestarts: supervisor.AlwaysRestart,
		Log: func(msg interface{}) {
			log.Debug("supervisor: ", msg)
		},
	}

	tasks := make(chan task.Request)

	svr.AddFunc(func(ctx context.Context) {
		runner.StartWorkers(ctx, rnr, tasks)
	})

	svr.AddFunc(func(ctx context.Context) {
		srv := server.New(ctx, ds, mq, rnr, tasks, server.DefaultEnqueue)
		srv.Run()
		<-ctx.Done()
	})

	apiURL := viper.GetString(envAPIURL)
	svr.AddFunc(func(ctx context.Context) {
		runner.RunAsyncRunner(ctx, apiURL, tasks, rnr)
	})

	svr.Serve(ctx)
	close(tasks)
}
