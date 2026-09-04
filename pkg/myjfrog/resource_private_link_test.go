package myjfrog_test

import (
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

func TestAccPrivateLink_full(t *testing.T) {
	jfrogURL := os.Getenv("JFROG_URL")
	if !strings.HasSuffix(jfrogURL, "jfrog.io") {
		t.Skipf("env var JFROG_URL '%s' is not a cloud instance. MyJFrog features are only available on cloud.", jfrogURL)
	}

	privateLinkID := os.Getenv("JFROG_MYJFROG_PRIVATE_LINK_ID")
	if privateLinkID == "" {
		t.Skipf("env var JFROG_MYJFROG_PRIVATE_LINK_ID is not set")
	}

	serverName := os.Getenv("JFROG_MYJFROG_PRIVATE_LINK_SERVER_NAME")
	if serverName == "" {
		t.Skipf("env var JFROG_MYJFROG_PRIVATE_LINK_SERVER_NAME is not set")
	}

	_, fqrn, resourceName := testutil.MkNames("test-myjfrog-private-link", "myjfrog_private_link")

	temp := `
	resource "myjfrog_private_link" "{{ .name }}" {
		private_link_id = "{{ .privateLinkId }}"
		server_names    = ["{{ .serverName }}"]
	}`

	testData := map[string]string{
		"name":          resourceName,
		"privateLinkId": privateLinkID,
		"serverName":    serverName,
	}

	config := util.ExecuteTemplate(resourceName, temp, testData)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "private_link_id", testData["privateLinkId"]),
					resource.TestCheckResourceAttr(fqrn, "server_names.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "server_names.0", testData["serverName"]),
					resource.TestCheckResourceAttr(fqrn, "servers.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "servers.0.server_name", testData["serverName"]),
					resource.TestCheckResourceAttrSet(fqrn, "servers.0.status"),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateId:                        testData["privateLinkId"],
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "private_link_id",
			},
		},
	})
}
