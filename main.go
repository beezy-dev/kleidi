/*
Forked from the KMSv2 mockup example
Source: https://github.com/kubernetes/kms.git
Apache 2.0 License
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

var (
	listenAddr      = flag.String("listen-addr", "unix:///tmp/kleidi.socket", "gRPC listen address.")
	timeout         = flag.Duration("timeout", 5*time.Second, "gRPC timeout.")
	providerService = flag.String("provider-service", "pkcs11", "KMS provider to connect to (pkcs11, vault).")
	configFilePath  = flag.String("config-file-path", "/opt/softhsm/config.json", "SoftHSM config file pat.")
	kleidiVersion string
)

func main() {

	log.Println("--------------------------------------------------------")
	log.Println("Kleidi", "v"+kleidiVersion, "KMS Provider Plugin for Kubernetes.")
	log.Println("License Apache 2.0 - https://github.com/beezy-dev/kleidi")
	log.Println("--------------------------------------------------------")

	// parsing environment variables
	flag.Parse()

	// defining the socket location
	log.Println("INFO: listen-addr flag set to:", *listenAddr)
	addr, err := util.ParseEndpoint(*listenAddr)
	if err != nil {
		log.Fatalln("EXIT: listen-addr flag failed with error:", err.Error())
	}

	// checking which provider to call
	log.Println("INFO: provider-service flag set to:", *providerService)
	providerServices := []string{"pkcs11", "vault"}
	if !slices.Contains(providerServices, *providerService) {
		log.Fatalln("EXIT: provider-service flag set to", *providerService, "is not supported. Refer to documentation for supported provider services.")
	}

	switch *providerService {
	case "pkcs11":
		// calling for the KMS services and checking connectivity
		log.Println("INFO: config-file-path flag set to:", *configFilePath)
		remoteKMSService, err := providers.NewPKCS11RemoteService(*configFilePath, "kleidi-kms")
		if err != nil {
			log.Fatalln("EXIT: config-file-path, set to", *configFilePath, ", failed with error:", err.Error())
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
				log.Fatalln("EXIT: failed to serve with error:", err.Error())
			}
		}()

		<-ctx.Done()
		grpcService.Shutdown()

	case "vault":
		log.Fatalln("EXIT: provider-service flag, set to", *providerService, ", is not yet implemented.")
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
