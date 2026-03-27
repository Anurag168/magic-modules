package workloadidentity_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-google/google/acctest"
)

func TestAccWorkloadIdentityServiceAgent_AllFieldsPresent(t *testing.T) {
	t.Parallel()

	acctest.VcrTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.AccTestPreCheck(t) },
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: testGoogleProjectServiceAgent_basic(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceAgentsFields("google_workload_identity_service_agent.service_agents", "data.google_project.project"),
				),
			},
		},
	})
}

func testGoogleProjectServiceAgent_basic() string {
	return `
data "google_project" "project" {}

resource "google_workload_identity_service_agent" "service_agents" {
  parent   = "projects/${data.google_project.project.number}/locations/global/serviceProducers/bigquery.googleapis.com"
}
`
}

// testAccCheckServiceAgentsFields checks if the serviceAgents array contains valid entries.
func testAccCheckServiceAgentsFields(resourceName, projectDataSourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		projRs, ok := s.RootModule().Resources[projectDataSourceName]
		if !ok {
			return fmt.Errorf("data source not found: %s", projectDataSourceName)
		}
		projectNumber := projRs.Primary.Attributes["number"]
		if projectNumber == "" {
			return fmt.Errorf("could not find project_number from data source %s", projectDataSourceName)
		}
		expectedContainer := "projects/" + projectNumber

		serviceAgentsCountStr, ok := rs.Primary.Attributes["service_agents.#"]
		if !ok {
			return fmt.Errorf("attribute 'service_agents' not found in resource %s", resourceName)
		}
		serviceAgentsCount, err := strconv.Atoi(serviceAgentsCountStr)
		if err != nil {
			return fmt.Errorf("error parsing service_agents count: %w", err)
		}

		if serviceAgentsCount == 0 {
			return fmt.Errorf("attribute 'service_agents' is empty in resource %s", resourceName)
		}

		for i := 0; i < serviceAgentsCount; i++ {
			prefix := fmt.Sprintf("service_agents.%d.", i)

			// 1. Check for 'container' field and project number
			containerKey := prefix + "container"
			containerValue, exists := rs.Primary.Attributes[containerKey]
			if !exists {
				return fmt.Errorf("field '%s' not found in service_agents[%d] for resource %s", "container", i, resourceName)
			}
			if containerValue != expectedContainer {
				return fmt.Errorf("unexpected container value '%s' in service_agents[%d], expected '%s'", containerValue, i, expectedContainer)
			}

			// 2. Check for 'service_producer' field
			serviceProducerKey := prefix + "service_producer"
			serviceProducerValue, exists := rs.Primary.Attributes[serviceProducerKey]
			if !exists {
				return fmt.Errorf("field '%s' not found in service_agents[%d] for resource %s", "service_producer", i, resourceName)
			}
			if serviceProducerValue != "bigquery.googleapis.com" {
				return fmt.Errorf("unexpected service_producer value '%s' in service_agents[%d], expected 'bigquery.googleapis.com'", serviceProducerValue, i)
			}

			// 3. Check for 'principal' field and "serviceAccount:" prefix
			principalKey := prefix + "principal"
			principalValue, exists := rs.Primary.Attributes[principalKey]
			if !exists {
				return fmt.Errorf("field '%s' not found in service_agents[%d] for resource %s", "principal", i, resourceName)
			}
			if !strings.HasPrefix(principalValue, "serviceAccount:") {
				return fmt.Errorf("principal value '%s' in service_agents[%d] does not start with 'serviceAccount:'", principalValue, i)
			}

			// 4. Check for 'role' field: if present, must start with "roles/"
			roleKey := prefix + "role"
			roleValue, hasRole := rs.Primary.Attributes[roleKey]
			if hasRole && roleValue != "" {
				if !strings.HasPrefix(roleValue, "roles/") {
					return fmt.Errorf("role value '%s' in service_agents[%d] does not start with 'roles/'", roleValue, i)
				}
			}
			// It's acceptable for role to be absent or empty.
		}
		return nil
	}
}
