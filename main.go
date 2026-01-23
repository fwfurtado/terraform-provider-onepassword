/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"context"
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
	opts := providerserver.ServeOpts{
		Address: defaultProviderAddress,
		Debug:   true,
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
