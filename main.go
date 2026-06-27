package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var (
	version string = "0.1.0"
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/latticeve/lattice",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), New(version), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
