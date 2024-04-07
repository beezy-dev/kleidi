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

const (
	socketTimeOut	= 5*time.Second
)

var kleidiVersion string

func main() {
	var (
		listenAddr      = flag.String("listen-addr", "unix:///tmp/kleidi/kleidi-kms-plugin.socket", "gRPC listen address.")
		timeout         = flag.Duration("timeout", socketTimeOut, "gRPC timeout.")
		providerService = flag.String("provider-service", "softhsm", "KMS provider to connect to (hvault, softhsm, TPM).")
		configFilePath  = flag.String("config-file-path", "/opt/softhsm/config.json", "SoftHSM config file pat.")
	)

	// parsing environment variables.
	flag.Parse()

	log.Println("--------------------------------------------------------")
	log.Println("Kleidi", "v"+kleidiVersion, "KMS Provider Plugin for Kubernetes.")
	log.Println("License Apache 2.0 - https://github.com/beezy-dev/kleidi")
	log.Println("--------------------------------------------------------")

	// defining the socket location.	
	addr, err := util.ParseEndpoint(*listenAddr)
	if err != nil {
		log.Fatalln("EXIT: listen-addr set to", *listenAddr, "failed with error:\n", err.Error())
	}

	log.Println("INFO: listen-addr set to", *listenAddr)

	// checking if a socket file already exists on the file system
	if cleanup := os.Remove(addr); cleanup != nil && !os.IsNotExist(cleanup) {
		log.Fatalln("EXIT: unable to delete existing socket file", addr, "from directory!")
	}

	// checking which provider to call.
	providerServices := []string{"softhsm", "hvault", "TPM"}
	if !slices.Contains(providerServices, *providerService) {
		log.Fatalln("EXIT: provider-service set to", *providerService, "is not supported.")
	}

	log.Println("INFO: provider-service set to", *providerService)

	switch *providerService {
	case "softhsm":
		// calling for the KMS services and checking connectivity.
		remoteKMSService, err := providers.NewPKCS11RemoteService(*configFilePath, "kleidi-kms-plugin")
		if err != nil {
			log.Fatalln("EXIT: config-file-path set to", *configFilePath, "failed with error:\n", err.Error())
		}

		log.Println("INFO: config-file-path set to", *configFilePath)

		// catch SIG termination.
		ctx := withShutdownSignal(context.Background())
		grpcService := service.NewGRPCService(
			addr,
			*timeout,
			remoteKMSService,
		)
		// starting service.
		go func() {
			if err := grpcService.ListenAndServe(); err != nil {
				log.Fatalln("EXIT: failed to serve with error:\n", err.Error())
			}
		}()

		<-ctx.Done()
		grpcService.Shutdown()

	case "hvault":
		log.Fatalln("EXIT: provider-service set to", *providerService, "is not yet implemented.")
	case "TPM":
		log.Fatalln("EXIT: provider-service set to", *providerService, "is not yet implemented.")
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