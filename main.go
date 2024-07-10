package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/happycakefriends/certainly/pkg/certainly"
	"github.com/happycakefriends/certainly/pkg/httpd"
	"github.com/happycakefriends/certainly/pkg/imapd"
	"github.com/happycakefriends/certainly/pkg/nameserver"
	"github.com/happycakefriends/certainly/pkg/notification"
	"github.com/happycakefriends/certainly/pkg/smtpd"

	"go.uber.org/zap"
)

func main() {

	configPtr := flag.String("c", "./config.cfg", "config file location")
	flag.Parse()
	// Read global config
	var err error
	var logger *zap.Logger
	config, usedConfigFile, err := certainly.ReadConfig(*configPtr)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	logger, err = certainly.SetupLogging(config)
	if err != nil {
		fmt.Printf("Could not set up logging: %s\n", err)
		os.Exit(1)
	}
	// Make sure to flush the zap logger buffer before exiting
	defer logger.Sync() //nolint:all
	sugar := logger.Sugar()

	sugar.Infow("Using config file",
		"file", usedConfigFile)
	sugar.Info("Starting up")
	// Error channel for servers
	errChan := make(chan error, 1)

	notifications := notification.Initialize(&config, sugar)

	dnsservers := nameserver.InitAndStart(&config, sugar, notifications, errChan)

	tlsconfig, err := setupTLS(dnsservers, &config, sugar)
	if err != nil {
		sugar.Fatalf("Could not start, error in creating TLS config",
			"error", err)
	}
	imaptlsconfig, err := setupTLS(dnsservers, &config, sugar)
	if err != nil {
		sugar.Fatalf("Could not start, error in creating TLS config",
			"error", err)
	}
	smtpd := smtpd.Initialize(&config, tlsconfig, sugar, notifications, errChan)
	imapd := imapd.Initialize(&config, imaptlsconfig, sugar, notifications, errChan)
	httpd.InitAndStart(&config, tlsconfig, sugar, notifications, errChan)
	smtpd.Start()
	imapd.Start()
	if err != nil {
		sugar.Error(err)
	}
	for {
		err = <-errChan
		if err != nil {
			sugar.Fatal(err)
		}
	}
}
