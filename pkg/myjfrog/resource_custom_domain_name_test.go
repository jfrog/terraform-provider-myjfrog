package myjfrog_test

import (
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

// To get this test works, set env vars with the path to the PEM files
// Private key file needs to be converted to RSA format:
// openssl rsa -in private.key -out private_rsa.key -traditional
func TestAccCustomDomainName_full(t *testing.T) {
	bodyPath := os.Getenv("JFROG_MYJFROG_CERTIFICATE_BODY_PATH")
	if bodyPath == "" {
		t.Skipf("env var JFROG_MYJFROG_CERTIFICATE_BODY_PATH is not set")
	}

	privateKeyPath := os.Getenv("JFROG_MYJFROG_CERTIFICATE_PRIVATE_KEY_PATH")
	if privateKeyPath == "" {
		t.Skipf("env var JFROG_MYJFROG_CERTIFICATE_PRIVATE_KEY_PATH is not set")
	}

	bodyBytes, err := os.ReadFile(bodyPath)
	if err != nil {
		t.Fatal(err)
	}

	privateKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		t.Fatal(err)
	}

	_, fqrn, resourceName := testutil.MkNames("test-custom-domain-name", "myjfrog_custom_domain_name")

	temp := `
	resource "myjfrog_custom_domain_name" "{{ .name }}" {
		certificate_name = "{{ .name }}"
		certificate_body = <<EOT
{{ .certificate_body }}EOT
		certificate_private_key = <<EOT
{{ .certificate_private_key }}EOT
		domains_under_certificate = [
			{
				url = "tf-myjfrog-test.jfrog.tech"
				server_name = "partnerenttest"
				type = "platform_base_url"
			}
		]
	}`

	testData := map[string]string{
		"name":                    resourceName,
		"certificate_body":        string(bodyBytes),
		"certificate_private_key": string(privateKeyBytes),
	}

	config := util.ExecuteTemplate(resourceName, temp, testData)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "certificate_name", testData["name"]),
					resource.TestCheckResourceAttr(fqrn, "certificate_body", testData["certificate_body"]),
					resource.TestCheckResourceAttr(fqrn, "certificate_private_key", testData["certificate_private_key"]),
					resource.TestCheckNoResourceAttr(fqrn, "certificate_chain"),
					resource.TestCheckResourceAttrSet(fqrn, "certificate_expiry"),
					resource.TestCheckResourceAttrSet(fqrn, "certificate_status"),
					resource.TestCheckResourceAttr(fqrn, "domains_under_certificate.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "domains_under_certificate.0.url", "tf-myjfrog-test.jfrog.tech"),
					resource.TestCheckResourceAttr(fqrn, "domains_under_certificate.0.server_name", "partnerenttest"),
					resource.TestCheckResourceAttr(fqrn, "domains_under_certificate.0.type", "platform_base_url"),
				),
			},
			{
				ResourceName:            fqrn,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"certificate_private_key"},
			},
		},
	})
}
func TestAccCustomDomainName_update(t *testing.T) {
	bodyPath := os.Getenv("JFROG_MYJFROG_CERTIFICATE_BODY_PATH")
	if bodyPath == "" {
		t.Skipf("env var JFROG_MYJFROG_CERTIFICATE_BODY_PATH is not set")
	}

	privateKeyPath := os.Getenv("JFROG_MYJFROG_CERTIFICATE_PRIVATE_KEY_PATH")
	if privateKeyPath == "" {
		t.Skipf("env var JFROG_MYJFROG_CERTIFICATE_PRIVATE_KEY_PATH is not set")
	}

	bodyBytes, err := os.ReadFile(bodyPath)
	if err != nil {
		t.Fatal(err)
	}

	privateKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		t.Fatal(err)
	}

	_, fqrn, resourceName := testutil.MkNames("test-custom-domain-name-update", "myjfrog_custom_domain_name")

	temp := `
	resource "myjfrog_custom_domain_name" "{{ .name }}" {
		certificate_name = "{{ .name }}"
		certificate_body = <<EOT
{{ .certificate_body }}EOT
		certificate_private_key = <<EOT
{{ .certificate_private_key }}EOT
		domains_under_certificate = [
			{
				url = "{{ .url }}"
				server_name = "partnerenttest"
				type = "platform_base_url"
			}
		]
	}`

	testData := map[string]string{
		"name":                    resourceName,
		"certificate_body":        string(bodyBytes),
		"certificate_private_key": string(privateKeyBytes),
		"url":                     "tf-myjfrog-test.jfrog.tech",
	}

	config := util.ExecuteTemplate(resourceName, temp, testData)

	testDataUpdated := map[string]string{
		"name":                    resourceName,
		"certificate_body":        string(bodyBytes),
		"certificate_private_key": string(privateKeyBytes),
		"url":                     "tf-myjfrog-test-updated.jfrog.tech",
	}

	configUpdated := util.ExecuteTemplate(resourceName, temp, testDataUpdated)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "certificate_name", testData["name"]),
					resource.TestCheckResourceAttr(fqrn, "domains_under_certificate.0.url", "tf-myjfrog-test.jfrog.tech"),
					resource.TestCheckResourceAttrSet(fqrn, "id"),
				),
			},
			{
				Config: configUpdated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "certificate_name", testDataUpdated["name"]),
					resource.TestCheckResourceAttr(fqrn, "domains_under_certificate.0.url", "tf-myjfrog-test-updated.jfrog.tech"),
					resource.TestCheckResourceAttrSet(fqrn, "id"),
				),
			},
		},
	})
}

func TestAccCustomDomainName_invalid_config(t *testing.T) {
	bodyPath := os.Getenv("JFROG_MYJFROG_CERTIFICATE_BODY_PATH")
	if bodyPath == "" {
		t.Skipf("env var JFROG_MYJFROG_CERTIFICATE_BODY_PATH is not set")
	}

	privateKeyPath := os.Getenv("JFROG_MYJFROG_CERTIFICATE_PRIVATE_KEY_PATH")
	if privateKeyPath == "" {
		t.Skipf("env var JFROG_MYJFROG_CERTIFICATE_PRIVATE_KEY_PATH is not set")
	}

	bodyBytes, err := os.ReadFile(bodyPath)
	if err != nil {
		t.Fatal(err)
	}

	privateKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		t.Fatal(err)
	}

	_, _, resourceName := testutil.MkNames("test-custom-domain-name", "myjfrog_custom_domain_name")

	temp := `
	resource "myjfrog_custom_domain_name" "{{ .name }}" {
		certificate_name = "{{ .name }}"
		certificate_body = <<EOT
{{ .certificate_body }}EOT
		certificate_private_key = <<EOT
{{ .certificate_private_key }}EOT
		domains_under_certificate = [
			{
				url = "tf-myjfrog-test.jfrog.tech"
				server_name = "partnerenttest"
				type = "docker_sub_domain"
			}
		]
	}`

	testData := map[string]string{
		"name":                    resourceName,
		"certificate_body":        string(bodyBytes),
		"certificate_private_key": string(privateKeyBytes),
	}

	config := util.ExecuteTemplate(resourceName, temp, testData)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`.*Expected 'docker_repository_name_override' to be configured when 'type' is.*`),
			},
		},
	})
}
