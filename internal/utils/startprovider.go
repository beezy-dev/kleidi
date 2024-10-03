package utils

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/beezy-dev/kleidi/internal/providers"
	"k8s.io/kms/pkg/service"
)

const (
	socketTimeOut = 10 * time.Second
)

func StartHvault(addr, providerConfig string, debug bool) {

	remoteKMSService, err := providers.NewVaultClientRemoteService(providerConfig, addr, debug)
	if err != nil {
		log.Fatalln("EXIT: remote HashiCorp Vault KMS provider failed with error:\n", err.Error())
	}

	ctx := withShutdownSignal(context.Background())
	grpcService := service.NewGRPCService(
		addr,
		socketTimeOut,
		remoteKMSService,
	)
	go func() {
		if err := grpcService.ListenAndServe(); err != nil {
			log.Fatalln("EXIT: failed to serve with error:\n", err.Error())
		}
	}()

	<-ctx.Done()
	grpcService.Shutdown()

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
