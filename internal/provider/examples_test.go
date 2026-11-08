package provider

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func TestExampleConfigurationsParse(t *testing.T) {
	server := providerserver.NewProtocol6(New("test")())()
	schemas, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	assertProtocolDiagnostics(t, schemas.Diagnostics)
	files, err := filepath.Glob("../../examples/*/*.tf")
	if err != nil || len(files) == 0 {
		t.Fatalf("example files = %v, error = %v", files, err)
	}
	for _, name := range files {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(name)
			if err != nil {
				t.Fatal(err)
			}
			file, diagnostics := hclsyntax.ParseConfig(data, name, hcl.Pos{Line: 1, Column: 1})
			if diagnostics.HasErrors() {
				t.Fatalf("invalid example: %s", diagnostics.Error())
			}
			if !bytes.Equal(data, hclwrite.Format(data)) {
				t.Error("example is not formatted in canonical HCL style")
			}
			for _, block := range file.Body.(*hclsyntax.Body).Blocks {
				var schema *tfprotov6.Schema
				switch block.Type {
				case "resource":
					if len(block.Labels) != 2 {
						t.Errorf("invalid resource block labels")
						continue
					}
					schema = schemas.ResourceSchemas[block.Labels[0]]
				case "data":
					if len(block.Labels) != 2 {
						t.Errorf("invalid data block labels")
						continue
					}
					schema = schemas.DataSourceSchemas[block.Labels[0]]
				default:
					continue
				}
				if schema == nil {
					t.Errorf("unknown %s type %q", block.Type, block.Labels[0])
					continue
				}
				attributes := make(map[string]*tfprotov6.SchemaAttribute)
				for _, attribute := range schema.Block.Attributes {
					attributes[attribute.Name] = attribute
				}
				for attribute := range block.Body.Attributes {
					if attr, ok := attributes[attribute]; !ok || (!attr.Optional && !attr.Required) {
						t.Errorf("invalid %s attribute %q", block.Labels[0], attribute)
					}
				}
				for name, attribute := range attributes {
					if _, ok := block.Body.Attributes[name]; attribute.Required && !ok {
						t.Errorf("missing required %s attribute %q", block.Labels[0], name)
					}
				}
			}
		})
	}
}
