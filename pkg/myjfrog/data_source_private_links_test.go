package myjfrog_test

import (
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

func TestAccPrivateLinks_all(t *testing.T) {
	jfrogURL := os.Getenv("JFROG_URL")
	if !strings.HasSuffix(jfrogURL, "jfrog.io") {
		t.Skipf("env var JFROG_URL '%s' is not a cloud instance. MyJFrog features are only available on cloud.", jfrogURL)
	}

	serverName := os.Getenv("JFROG_MYJFROG_PRIVATE_LINK_SERVER_NAME")
	if serverName == "" {
		t.Skipf("env var JFROG_MYJFROG_PRIVATE_LINK_SERVER_NAME is not set")
	}

	_, fqrn, resourceName := testutil.MkNames("test-myjfrog-private-links", "myjfrog_private_links")

	temp := `
	data "myjfrog_private_links" "{{ .name }}" {
		server_name = "{{ .serverName }}"
	}`

	testData := map[string]string{
		"name":       resourceName,
		"serverName": serverName,
	}

	config := util.ExecuteTemplate(resourceName, temp, testData)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "server_name", testData["serverName"]),
					resource.TestCheckResourceAttrSet(fqrn, "private_links.#"),
				),
			},
		},
	})
}
