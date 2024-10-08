package myjfrog_test

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

func TestAccIPAllowlist_full(t *testing.T) {
	jfrogURL := os.Getenv("JFROG_URL")
	if !strings.HasSuffix(jfrogURL, "jfrog.io") {
		t.Skipf("env var JFROG_URL '%s' is not a cloud instance. MyJFrog features are only available on cloud.", jfrogURL)
	}

	_, fqrn, allowlistName := testutil.MkNames("test-myjfrog-ip-allowlist", "myjfrog_ip_allowlist")

	re := regexp.MustCompile(`^https://(\w+)\.jfrog\.io$`)
	matches := re.FindStringSubmatch(jfrogURL)
	if len(matches) < 2 {
		t.Fatalf("can't find server name from JFROG_URL %s", jfrogURL)
	}
	serverName := matches[1]

	temp := `
	resource "myjfrog_ip_allowlist" "{{ .name }}" {
		server_name = "{{ .serverName }}"
		ips = {{ .ips }}
	}`

	testData := map[string]string{
		"name":       allowlistName,
		"serverName": serverName,
		"ips":        `["1.1.1.7", "2.2.2.7/1"]`,
	}

	config := util.ExecuteTemplate(allowlistName, temp, testData)

	updatedTestData := map[string]string{
		"name":       allowlistName,
		"serverName": serverName,
		"ips":        `["2.2.2.7/1", "3.3.3.7/1"]`,
	}
	updatedConfig := util.ExecuteTemplate(allowlistName, temp, updatedTestData)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "server_name", testData["serverName"]),
					resource.TestCheckResourceAttr(fqrn, "ips.#", "2"),
					resource.TestCheckTypeSetElemAttr(fqrn, "ips.*", "1.1.1.7"),
					resource.TestCheckTypeSetElemAttr(fqrn, "ips.*", "2.2.2.7/1"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "server_name", updatedTestData["serverName"]),
					resource.TestCheckResourceAttr(fqrn, "ips.#", "2"),
					resource.TestCheckTypeSetElemAttr(fqrn, "ips.*", "2.2.2.7/1"),
					resource.TestCheckTypeSetElemAttr(fqrn, "ips.*", "3.3.3.7/1"),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateId:                        testData["serverName"],
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "server_name",
			},
		},
	})
}

func TestAccIPAllowlist_empty_ips(t *testing.T) {
	jfrogURL := os.Getenv("JFROG_URL")
	if !strings.HasSuffix(jfrogURL, "jfrog.io") {
		t.Skipf("env var JFROG_URL '%s' is not a cloud instance. MyJFrog features are only available on cloud.", jfrogURL)
	}

	_, fqrn, allowlistName := testutil.MkNames("test-myjfrog-ip-allowlist", "myjfrog_ip_allowlist")

	re := regexp.MustCompile(`^https://(\w+)\.jfrog\.io$`)
	matches := re.FindStringSubmatch(jfrogURL)
	if len(matches) < 2 {
		t.Fatalf("can't find server name from JFROG_URL %s", jfrogURL)
	}
	serverName := matches[1]

	temp := `
	resource "myjfrog_ip_allowlist" "{{ .name }}" {
		server_name = "{{ .serverName }}"
		ips = []
	}`

	testData := map[string]string{
		"name":       allowlistName,
		"serverName": serverName,
	}

	config := util.ExecuteTemplate(allowlistName, temp, testData)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "server_name", testData["serverName"]),
					resource.TestCheckResourceAttr(fqrn, "ips.#", "0"),
				),
			},
		},
	})
}

func TestAccIPAllowlist_invalid_ips(t *testing.T) {
	for _, invalidIP := range []string{"999.999.999.999", "invalid", "1.1.1.1/99", "999.2.2.2/1"} {
		t.Run(invalidIP, func(t *testing.T) {
			_, _, allowlistName := testutil.MkNames("test-myjfrog-ip-allowlist", "myjfrog_ip_allowlist")

			temp := `
			resource "myjfrog_ip_allowlist" "{{ .name }}" {
				server_name = "test-server"
				ips = ["{{ .invalidIPs }}"]
			}`

			testData := map[string]string{
				"name":       allowlistName,
				"invalidIPs": invalidIP,
			}

			config := util.ExecuteTemplate(allowlistName, temp, testData)

			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProviders(),
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile(`Invalid IP\/CIDR format`),
					},
				},
			})
		})
	}
}
