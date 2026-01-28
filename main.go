/*
Copyright © 2026 Fernando Furtado <fwfurtado@gmail.com>
*/
package main

import (
	"context"
	"flag"
	"log"

	onepasswordprovider "github.com/fwfurtado/onepassword-tf-provider/internal/onepassword/terraform/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

const (
			defaultProviderAddress = "registry.terraform.io/fwfurtado/onepassword"
)

var version string = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "Enable provider debug mode.")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: defaultProviderAddress,
		Debug:   debug,
	}

	err := providerserver.Serve(
		context.Background(),
		func() provider.Provider { return onepasswordprovider.New(version) },
		opts,
	)

	if err != nil {
		log.Fatalf("Error serving provider: %v", err)
	}
}






