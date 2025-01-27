package buildkite

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccBuildkiteClusterQueueResource(t *testing.T) {
	configBasic := func(fields ...string) string {
		return fmt.Sprintf(`
		provider "buildkite" {
			timeouts {
				create = "10s"
				read = "10s"
				update = "10s"
				delete = "10s"
			}
		}

		resource "buildkite_cluster" "cluster_test" {
			name = "Test cluster %s"
			description = "Acceptance testing cluster"
		}

		resource "buildkite_cluster_queue" "foobar" {
			cluster_id = buildkite_cluster.cluster_test.id
			key = "queue-%s"
			description = "Acceptance test %s"
		}
		`, fields[0], fields[1], fields[2])
	}

	t.Run("creates a cluster queue", func(t *testing.T) {
		var cq ClusterQueueResourceModel
		clusterName := acctest.RandString(10)
		queueKey := acctest.RandString(10)
		queueDesc := acctest.RandString(10)

		check := resource.ComposeAggregateTestCheckFunc(
			// Confirm the cluster queue exists in the buildkite API
			testAccCheckClusterQueueExists("buildkite_cluster_queue.foobar", &cq),
			// Confirm the cluster queue has the correct values in Buildkite's system
			testAccCheckClusterQueueRemoteValues(&cq, fmt.Sprintf("Acceptance test %s", queueDesc), fmt.Sprintf("queue-%s", queueKey)),
			// Confirm the cluster queue has the correct values in terraform state
			resource.TestCheckResourceAttr("buildkite_cluster_queue.foobar", "key", fmt.Sprintf("queue-%s", queueKey)),
			resource.TestCheckResourceAttr("buildkite_cluster_queue.foobar", "description", fmt.Sprintf("Acceptance test %s", queueDesc)),
			resource.TestCheckResourceAttrSet("buildkite_cluster_queue.foobar", "id"),
			resource.TestCheckResourceAttrSet("buildkite_cluster_queue.foobar", "uuid"),
		)

		resource.ParallelTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: protoV6ProviderFactories(),
			CheckDestroy:             testAccCheckClusterQueueDestroy,
			Steps: []resource.TestStep{
				{
					Config: configBasic(clusterName, queueKey, queueDesc),
					Check:  check,
				},
				{
					RefreshState: true,
					PlanOnly:     true,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttrSet("buildkite_cluster_queue.foobar", "key"),
						resource.TestCheckResourceAttrSet("buildkite_cluster_queue.foobar", "description"),
					),
				},
			},
		})
	})

	t.Run("updates a cluster queue", func(t *testing.T) {
		var cq ClusterQueueResourceModel
		clusterName := acctest.RandString(10)
		queueKey := acctest.RandString(10)
		queueDesc := acctest.RandString(10)
		updatedQueueDesc := acctest.RandString(10)

		check := resource.ComposeAggregateTestCheckFunc(
			// Confirm the cluster queue exists in the buildkite API
			testAccCheckClusterQueueExists("buildkite_cluster_queue.foobar", &cq),
			// Confirm the cluster queue has the correct values in Buildkite's system
			testAccCheckClusterQueueRemoteValues(&cq, fmt.Sprintf("Acceptance test %s", queueDesc), fmt.Sprintf("queue-%s", queueKey)),
			// Confirm the cluster queue has the correct values in terraform state
			resource.TestCheckResourceAttr("buildkite_cluster_queue.foobar", "key", fmt.Sprintf("queue-%s", queueKey)),
			resource.TestCheckResourceAttr("buildkite_cluster_queue.foobar", "description", fmt.Sprintf("Acceptance test %s", queueDesc)),
		)

		ckecUpdated := resource.ComposeAggregateTestCheckFunc(
			// Confirm the cluster queue exists in the buildkite API
			testAccCheckClusterQueueExists("buildkite_cluster_queue.foobar", &cq),
			// Confirm the cluster queue has the correct values in Buildkite's system
			testAccCheckClusterQueueRemoteValues(&cq, fmt.Sprintf("Acceptance test %s", updatedQueueDesc), fmt.Sprintf("queue-%s", queueKey)),
			// Confirm the cluster queue has the correct values in terraform state
			resource.TestCheckResourceAttr("buildkite_cluster_queue.foobar", "key", fmt.Sprintf("queue-%s", queueKey)),
			resource.TestCheckResourceAttr("buildkite_cluster_queue.foobar", "description", fmt.Sprintf("Acceptance test %s", updatedQueueDesc)),
		)

		resource.ParallelTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: protoV6ProviderFactories(),
			CheckDestroy:             testAccCheckClusterQueueDestroy,
			Steps: []resource.TestStep{
				{
					Config: configBasic(clusterName, queueKey, queueDesc),
					Check:  check,
				},
				{
					Config: configBasic(clusterName, queueKey, updatedQueueDesc),
					Check:  ckecUpdated,
				},
			},
		})
	})

	t.Run("imports a cluster queue", func(t *testing.T) {
		var cq ClusterQueueResourceModel
		clusterName := acctest.RandString(10)
		queueKey := acctest.RandString(10)
		queueDesc := acctest.RandString(10)

		check := resource.ComposeAggregateTestCheckFunc(
			// Confirm the cluster queue exists in the buildkite API
			testAccCheckClusterQueueExists("buildkite_cluster_queue.foobar", &cq),
			// Check to confirm the local state is correct before we re-import it
			resource.TestCheckResourceAttr("buildkite_cluster_queue.foobar", "key", fmt.Sprintf("queue-%s", queueKey)),
			resource.TestCheckResourceAttr("buildkite_cluster_queue.foobar", "description", fmt.Sprintf("Acceptance test %s", queueDesc)),
		)

		resource.ParallelTest(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: protoV6ProviderFactories(),
			CheckDestroy:             testAccCheckClusterQueueDestroy,
			Steps: []resource.TestStep{
				{
					Config: configBasic(clusterName, queueKey, queueDesc),
					Check:  check,
				},
				{
					// re-import the resource (using the graphql token of the existing resource) and confirm they match
					ResourceName:      "buildkite_cluster_queue.foobar",
					ImportStateIdFunc: testAccGetImportClusterQueueId(&cq),
					ImportState:       true,
					ImportStateVerify: true,
				},
			},
		})
	})
}

func testAccCheckClusterQueueExists(resourceName string, clusterQueueResourceModel *ClusterQueueResourceModel) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]

		if !ok {
			return fmt.Errorf("Not found in state: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("No ID is set in state")
		}

		// Obtain queues of the queue's cluster from its cluster UUID
		queues, err := getClusterQueues(
			context.Background(),
			genqlientGraphql,
			getenv("BUILDKITE_ORGANIZATION_SLUG"),
			resourceState.Primary.Attributes["cluster_uuid"],
		)

		// If cluster queues were not able to be fetched by Genqlient
		if err != nil {
			return fmt.Errorf("Error fetching Cluster queues from graphql API: %v", err)
		}

		// Obtain the ClusterQueueResourceModel from the queues slice
		for _, edge := range queues.Organization.Cluster.Queues.Edges {
			if edge.Node.Id == resourceState.Primary.ID {
				updateClusterQueueResource(edge.Node, clusterQueueResourceModel)
				break
			}
		}

		// If clusterQueueResourceModel isnt set from the queues slice
		if clusterQueueResourceModel.Id.ValueString() == "" {
			return fmt.Errorf("No Cluster queue found with graphql id: %s", resourceState.Primary.ID)
		}

		return nil
	}
}

func testAccCheckClusterQueueRemoteValues(cq *ClusterQueueResourceModel, description, key string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if cq.Key.ValueString() != key {
			return fmt.Errorf("Remote Cluster queue key (%s) doesn't match expected value (%s)", cq.Key, key)
		}

		if cq.Description.ValueString() != description {
			return fmt.Errorf("Remote Cluster queue description (%s) doesn't match expected value (%s)", cq.Description, description)
		}

		return nil
	}
}

func testAccGetImportClusterQueueId(cq *ClusterQueueResourceModel) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		// Obtain trimmed cluster ID and cluster UUID
		clusterUuid := strings.Trim(cq.Id.ValueString(), "\"")
		clusterQueueID := strings.Trim(cq.ClusterUuid.ValueString(), "\"")
		// Set ID for import
		id := fmt.Sprintf("%s,%s", clusterUuid, clusterQueueID)
		return id, nil
	}
}

func testAccCheckClusterQueueDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "buildkite_cluster_queue" {
			continue
		}
		// Try to obtain the queues' cluster by its ID
		resp, err := getNode(context.Background(), genqlientGraphql, rs.Primary.Attributes["cluster_id"])
		// If exists a getNodeNodeCluster, cluster still exists, error
		if clusterNode, ok := resp.GetNode().(*getNodeNodeCluster); ok {
			// Cluster still exists
			return fmt.Errorf("Cluster still exists: %v", clusterNode)
		}
		return err
	}
	return nil
}
