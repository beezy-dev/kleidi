package providers

import (
	"log"
	"time"
)

func startProvider(addr, provider, providerConfig string, socketTimeOut time.Duration) {
	log.Println(addr, provider, providerConfig, socketTimeOut)
	// switch provider {
	// case "softhsm":
	// 	_, err := startSofthsm(addr, provider, providerConfig)
	// 	if err != nil {
	// 		fmt.Errorf("test")
	// 	}
	// }
}

// func startSofthsm(addr, provider, providerConfig string, socketTimeOut time.Duration) {

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
// }

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
