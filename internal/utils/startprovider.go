package utils

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/beezy-dev/kleidi/internal/providers"
	"k8s.io/kms/pkg/service"
	"go.uber.org/zap"
)

const (
	socketTimeOut = 10 * time.Second
)

func StartProvider(addr, provider, providerConfig string, debug bool) {

	switch provider {
	case "softhsm":
		startSofthsm(addr, provider, providerConfig, debug)
	case "hvault":
		startHvault(addr, provider, providerConfig, debug)
	case "tpm":
		startTpm(addr, provider, providerConfig, debug)
	}
}

func startSofthsm(addr, provider, providerConfig string, debug bool) {

	remoteKMSService, err := providers.NewPKCS11RemoteService(providerConfig, "kleidi-kms-plugin")
	if err != nil {
		zap.L().Fatal("EXIT: remote KMS provider [" + provider + "] failed with error: " + err.Error())
	}
	// catch SIG termination.
	ctx := withShutdownSignal(context.Background())
	grpcService := service.NewGRPCService(
		addr,
		socketTimeOut,
		remoteKMSService,
	)
	// starting service.
	go func() {
		if err := grpcService.ListenAndServe(); err != nil {
			zap.L().Fatal("EXIT: failed to serve with error: " + err.Error())
		}
	}()

	<-ctx.Done()
	grpcService.Shutdown()
}

func startHvault(addr, provider, providerConfig string, debug bool) {

	remoteKMSService, err := providers.NewVaultClientRemoteService(providerConfig, addr, debug)
	if err != nil {
		zap.L().Fatal("EXIT: remote KMS provider [" + provider + "] failed with error: " + err.Error())
	}

	ctx := withShutdownSignal(context.Background())
	grpcService := service.NewGRPCService(
		addr,
		socketTimeOut,
		remoteKMSService,
	)
	go func() {
		if err := grpcService.ListenAndServe(); err != nil {
			zap.L().Fatal("EXIT: failed to serve with error: " + err.Error())
		}
	}()

	<-ctx.Done()
	grpcService.Shutdown()

}

func startTpm(addr, provider, providerConfig string, debug bool) {

	zap.L().Info("BETA: flag -provider " + provider + " with -listen " + addr + " and -configfile " + providerConfig + " currently unsafe to be used in production.")
	providers.TmpPlaceholder()

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
