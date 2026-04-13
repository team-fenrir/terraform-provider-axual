package ApplicationPrincipalResource

import (
	. "axual.com/terraform-provider-axual/internal/tests"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestApplicationPrincipalDeleteResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			// Setup: create two principals so deletion of one is allowed
			{
				Config: GetProvider() + GetFile(
					"axual_application_principal_delete_setup.tf",
					"axual_application_principal_delete_two_principals.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile("axual_application_principal.delete_principal_1", "principal", "certs/generic_application_1.cer"),
					CheckBodyMatchesFile("axual_application_principal.delete_principal_2", "principal", "certs/generic_application_2.cer"),
				),
			},
			// Verify successful deletion when multiple principals exist
			{
				Config: GetProvider() + GetFile(
					"axual_application_principal_delete_setup.tf",
					"axual_application_principal_delete_one_principal.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile("axual_application_principal.delete_principal_1", "principal", "certs/generic_application_1.cer"),
				),
			},
			// Verify error when attempting to delete the last principal
			{
				Config: GetProvider() + GetFile(
					"axual_application_principal_delete_setup.tf",
				),
				ExpectError: regexp.MustCompile("Cannot delete the last application principal"),
			},
			// Cleanup
			{
				Destroy: true,
				Config: GetProvider() + GetFile(
					"axual_application_principal_delete_setup.tf",
					"axual_application_principal_delete_one_principal.tf",
				),
			},
		},
	})
}

func TestApplicationPrincipalConnectorDeleteResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			// Setup: create two principals on a connector so deletion of one is allowed
			{
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_delete_two_principals.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile("axual_application_principal.connector_delete_principal_1", "principal", "certs/generic_application_1.cer"),
					CheckBodyMatchesFile("axual_application_principal.connector_delete_principal_2", "principal", "certs/generic_application_2.cer"),
				),
			},
			// Verify successful deletion with connector stop/start behaviour
			{
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_delete_one_principal.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile("axual_application_principal.connector_delete_principal_1", "principal", "certs/generic_application_1.cer"),
				),
			},
			// Verify error when attempting to delete the last principal on a connector
			{
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
				),
				ExpectError: regexp.MustCompile("Cannot delete the last application principal"),
			},
			// Cleanup
			{
				Destroy: true,
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_delete_one_principal.tf",
				),
			},
		},
	})
}