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

	"github.com/beezy-dev/kleidi/internal/utils"
)

var kleidiVersion string

func main() {

	// Generic vars considering the consistency across providers.
	var (
		listenAddr         = flag.String("listen", "unix:///tmp/kleidi/kleidi-kms-plugin.socket", "gRPC listen address")
		providerConfigFile = flag.String("configfile", "/opt/kleidi/config.json", "Provider config file pat")
		debugMode          = flag.Bool("debugmode", false, "Enable debug mode")
	)

	// Parsing environment variables.
	flag.Parse()

	// Prettyfy the starting header fetching built version at compile time.
	log.Println("--------------------------------------------------------")
	log.Println("Kleidi", "v"+kleidiVersion, "for HashiCorp Vault Provider Plugin for Kubernetes.")
	log.Println("License Apache 2.0 - https://github.com/beezy-dev/kleidi")
	log.Println("--------------------------------------------------------")

	// Validating the socket location.
	addr, err := utils.ValidateListenAddr(*listenAddr)
	if err != nil {
		log.Fatalln("EXIT: flag -listen set to", *listenAddr, "failed with error:\n", err.Error())
	}

	// Checking and cleaning an existing socket in case of ungraceful shutdown.
	if cleanup := os.Remove(addr); cleanup != nil && !os.IsNotExist(cleanup) {
		log.Fatalln("EXIT: unable to delete existing socket file", addr, "from directory!")
	}

	// Validating the provider config.
	providerConfig, err := utils.ValidateConfigfile(*providerConfigFile)
	if err != nil {
		log.Fatalln("EXIT: flag -configfile set to", providerConfig, "failed with error:\n", err.Error())
	}

	debug := *debugMode

	//Starting the appropriate provider once previously validated.
	//REFACTOR to a simple interface

	utils.StartHvault(addr, providerConfig, debug)

}
