/*
Copyright 2023 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/beezy-dev/kleidi/providers"
	"k8s.io/kms/pkg/service"
	"k8s.io/kms/pkg/util"
)

const (
	kleidVersion = "0.1"
)

var (
	listenAddr      = flag.String("listen-addr", "unix:///tmp/kms.socket", "gRPC listen address.")
	timeout         = flag.Duration("timeout", 5*time.Second, "gRPC timeout.")
	providerService = flag.String("provider-service", "pkcs11", "KMS provider to connect to (pkcs11, vault).")
	configFilePath  = flag.String("config-file-path", "/etc/softhsm-config.json", "SoftHSM config file pat.")
)

func main() {

	log.Println("--------------------------------------------------------")
	log.Println("Kleidi", "v"+kleidVersion, "KMS Provider Plugin for Kubernetes.")
	log.Println("License Apache 2.0 - https://github.com/beezy-dev/kleidi")
	log.Println("--------------------------------------------------------")

	// parsing environment variables
	flag.Parse()

	// defining the socket location
	log.Println("Info: endpoint location defined as:", *listenAddr)
	addr, err := util.ParseEndpoint(*listenAddr)
	if err != nil {
		log.Fatalln("Fatal: failed to parse endpoint:", err.Error())
	}

	// checking which provider to call
	providerServices := []string{"pkcs11", "vault"}
	if !slices.Contains(providerServices, *providerService) {
		log.Fatalln("Fatal:", providerService, "is not supported. Refer to documentation for supported provider services.")
	}

	switch *providerService {
	case "pkcs11":
		log.Println("Info: Provider is set to:", *providerService)

		// calling for the KMS services and checking connectivity
		log.Println("Info: configuration file location defined as:", *configFilePath)
		remoteKMSService, err := providers.NewPKCS11RemoteService(*configFilePath, "kms-test")
		if err != nil {
			log.Fatalln("Fatal: failed to create remote service with error:", err.Error())
		}

		// catch SIG termination
		ctx := withShutdownSignal(context.Background())
		grpcService := service.NewGRPCService(
			addr,
			*timeout,
			remoteKMSService,
		)
		// starting service
		go func() {
			if err := grpcService.ListenAndServe(); err != nil {
				log.Fatalln("Fatal: failed to serve:", err.Error())
			}
		}()

		<-ctx.Done()
		grpcService.Shutdown()

	case "vault":
		log.Println("Info: Provider is set to:", *providerService)
		log.Fatalln("Fatal: Provider is not yet implemented.")
	}
}

// withShutdownSignal returns a copy of the parent context that will close if
// the process receives termination signals.
func withShutdownSignal(ctx context.Context) context.Context {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT, os.Interrupt)

	nctx, cancel := context.WithCancel(ctx)

	go func() {
		<-signalChan
		cancel()
	}()
	return nctx
}
