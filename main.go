/*
Forked from the KMSv2 mockup example
Source: https://github.com/kubernetes/kms.git
Apache 2.0 License
*/

package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/beez-dev/kleidi/utils"
)

const (
	socketTimeOut = 10 * time.Second
)

var kleidiVersion string

func main() {

	// Generic vars considering the consistency across providers.
	var (
		listenAddr         = flag.String("listen-addr", "unix:///tmp/kleidi/kleidi-kms-plugin.socket", "gRPC listen address.")
		providerService    = flag.String("provider-service", "softhsm", "KMS provider to connect to (hvault, softhsm, tpm).")
		providerConfigFile = flag.String("provider-config-file", "/opt/softhsm/config.json", "Provider config file pat.")
	)

	// Parsing environment variables.
	flag.Parse()

	// Prettyfy the starting header fetching built version at compile time.
	log.Println("--------------------------------------------------------")
	log.Println("Kleidi", "v"+kleidiVersion, "KMS Provider Plugin for Kubernetes.")
	log.Println("License Apache 2.0 - https://github.com/beezy-dev/kleidi")
	log.Println("--------------------------------------------------------")

	// Validating the socket location.
	addr, err := utils.ValidateListenAddr(*listenAddr)
	if err != nil {
		log.Fatalln("EXIT: listen-addr set to", *listenAddr, "failed with error:\n", err.Error())
	}

	// Checking and cleaning an existing socket in case of ungraceful shutdown.
	if cleanup := os.Remove(addr); cleanup != nil && !os.IsNotExist(cleanup) {
		log.Fatalln("EXIT: unable to delete existing socket file", addr, "from directory!")
	}

	// Validating the provider.
	provider, err := utils.ValidateProvider(*providerService)
	if err != nil {
		log.Fatalln("EXIT: provider-service set to", provider, "failed with error:\n", err.Error())
	}

	// Validating the provider config.
	providerConfig, err := utils.ValidateConfigfile(*providerConfigFile)
	if err != nil {
		log.Fatalln("EXIT: provider-config-file set to", providerConfig, "failed with error:\n", err.Error())
	}

	startKMS, err := providers.startProvider(addr, provider, providerConfig, socketTimeOut)
	if err != nil {
		log.Fatalln("EXIT: provider-config-file set to", providerConfig, "failed with error:\n", err.Error())
	}
	log.Println(startKMS)

	// Starting the appropriate provider once previously validated.
	// REFACTOR to a simple interface
	// switch provider {
	// case "softhsm":

	// 	// calling for the KMS services and checking connectivity.
	// 	remoteKMSService, err := providers.NewPKCS11RemoteService(providerConfig, "kleidi-kms-plugin")
	// 	if err != nil {
	// 		log.Fatalln("EXIT: remote KMS service set to", provider, "failed with error:\n", err.Error())
	// 	}

	// 	// catch SIG termination.
	// 	ctx := withShutdownSignal(context.Background())
	// 	grpcService := service.NewGRPCService(
	// 		addr,
	// 		socketTimeOut,
	// 		remoteKMSService,
	// 	)
	// 	// starting service.
	// 	go func() {
	// 		if err := grpcService.ListenAndServe(); err != nil {
	// 			log.Fatalln("EXIT: failed to serve with error:\n", err.Error())
	// 		}
	// 	}()

	// 	<-ctx.Done()
	// 	grpcService.Shutdown()

	// case "hvault":
	// 	log.Fatalln("EXIT: provider-service set to", provider, "is not yet implemented.")
	// case "tpm":
	// 	log.Fatalln("EXIT: provider-service set to", provider, "is not yet implemented.")
	// }
}

// // withShutdownSignal returns a copy of the parent context that will close if
// // the process receives termination signals.
// func withShutdownSignal(ctx context.Context) context.Context {
// 	signalChan := make(chan os.Signal, 1)
// 	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT, os.Interrupt)

// 	nctx, cancel := context.WithCancel(ctx)

// 	go func() {
// 		<-signalChan
// 		cancel()
// 	}()

// 	return nctx
// }
