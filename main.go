package main

import (
	"context"
	"flag"
	"log"

	"github.com/QDivZero/qdivzero-terraform/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var version = "dev"

func main() {
	debug := flag.Bool("debug", false, "enable provider debugging")
	flag.Parse()
	if err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/qdivzero/qdivzero",
		Debug:   *debug,
	}); err != nil {
		log.Fatal(err)
	}
}
