package myjfrog

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
	"github.com/samber/lo"
)

var _ resource.Resource = (*customDomainNameResource)(nil)

type customDomainNameResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
	Client       *resty.Client
}

func NewCustomDomainNameResource() resource.Resource {
	return &customDomainNameResource{
		TypeName: "myjfrog_custom_domain_name",
	}
}

func (r *customDomainNameResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *customDomainNameResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)

	if r.ProviderData.Client == nil {
		resp.Diagnostics.AddError(
			"Client not configured in provider",
			"MyJFrog Resty client is not configured due to missing `api_token` attribute in provider configuration, or missing `JFROG_MYJFROG_API_TOKEN` env var.",
		)
	}

	r.Client = r.ProviderData.Client.Clone().
		SetRetryCount(10).
		SetRetryWaitTime(1 * time.Minute).
		SetRetryMaxWaitTime(2 * time.Minute)
}

func (r *customDomainNameResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"certificate_name": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				MarkdownDescription: "Give your certificate a unique name to help identify it.",
			},
			"certificate_body": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				MarkdownDescription: "A certificate body is a text field containing information about the certificate such as the domain name, public key, signature algorithm, issuer, and validity period. You may include the certificate chain in the certificate body.",
			},
			"certificate_chain": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				MarkdownDescription: "A chain of trust between certificates that is anchored by a root certificate authority is used to verify the validity of a certificate.",
			},
			"certificate_private_key": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				MarkdownDescription: "The private key pairs with the certificate's public key for encryption. The keys must match. Private key must be in RSA format (RSA PKCS#1). If not, you can convert it to RSA using the following OpenSSL command:\n`openssl rsa -in private.key -out private_rsa.key -traditional`",
			},
			"certificate_status": schema.StringAttribute{
				Computed: true,
			},
			"certificate_expiry": schema.Int64Attribute{
				Computed: true,
			},
			"domains_under_certificate": schema.SetNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"url": schema.StringAttribute{
							Required: true,
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},
						"server_name": schema.StringAttribute{
							Required: true,
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
							MarkdownDescription: "The server_name field in your request refers to the designated JFrog server where you intend to configure a CNAME (Canonical Name). For instance, if your JFrog server URL is myserver.jfrog.io, and you wish to set up a CNAME for this server, you should enter `myserver` in the server_name field.",
						},
						"type": schema.StringAttribute{
							Required: true,
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
								stringvalidator.OneOf("platform_base_url", "docker_sub_domain"),
							},
							MarkdownDescription: "This attribute offers two valid options:\n" +
								"  * `platform_base_url`: Use this option when configuring the base URL for accessing the JPD (JFrog Platform Distribution) User Interface. You can also employ this URL as the base for API calls.\n" +
								"  * `docker_sub_domain`: Use this option when configuring the base URL for Docker-related operations, such as login, pull, and push actions, within the JPD. Please note that this parameter is exclusively available for JPD instances that have the Docker Subdomain method configured.\n" +
								"For more detailed information on the various Docker methods available in Artifactory, please refer to [Get Started With Artifactory as a Docker Registry](https://jfrog.com/help/r/jfrog-artifactory-documentation/get-started-with-artifactory-as-a-docker-registry).",
						},
						"docker_repository_name_override": schema.StringAttribute{
							Optional: true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(3, 32),
								stringvalidator.RegexMatches(regexp.MustCompile(`[a-z0-9]-`), "must only contain lowercase letters, numbers, and hyphens (-)"),
							},
							MarkdownDescription: "Only available when `type` is set to `docker_sub_domain` (when configuring the base URL for Docker-related operations). Use this option when you wish to define an explicit repository name (not derived from the domain name) -  please notice this is not best practice.",
						},
					},
				},
				Required: true,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
		},
		MarkdownDescription: "Provides a MyJFrog [Custom Domain Name](https://jfrog.com/help/r/jfrog-hosting-models-documentation/manage-custom-domain-names-in-myjfrog) resource to manage custom domain names. " +
			"Also see [Custom Domain Name REST API](https://jfrog.com/help/r/jfrog-rest-apis/custom-domain-name-rest-apis) for more details.\n\n" +
			"To use this resource, you need an access token. Only a Primary Admin can generate MyJFrog tokens. For more information, see [Generate a Token in MyJFrog](https://jfrog.com/help/r/jfrog-hosting-models-documentation/generate-a-token-in-myjfrog).",
	}
}

func (r customDomainNameResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data customDomainNameResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for idx, elem := range data.DomainsUnderCertificate.Elements() {
		attr := elem.(types.Object).Attributes()

		t := attr["type"].(types.String).ValueString()
		override, ok := attr["docker_repository_name_override"].(types.String)

		if t == "docker_sub_domain" && (!ok || override.IsNull()) {
			resp.Diagnostics.AddAttributeError(
				path.Root("domains_under_certificate").AtListIndex(idx).AtName("docker_repository_name_override"),
				"Invalid Attribute Configuration",
				"Expected 'docker_repository_name_override' to be configured when 'type' is set to 'docker_sub_domain'.",
			)
			return
		}
	}
}

type customDomainNameResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	CertificateName         types.String `tfsdk:"certificate_name"`
	CertificateBody         types.String `tfsdk:"certificate_body"`
	CertificateChain        types.String `tfsdk:"certificate_chain"`
	CertificatePrivateKey   types.String `tfsdk:"certificate_private_key"`
	CertificateStatus       types.String `tfsdk:"certificate_status"`
	CertificateExpiry       types.Int64  `tfsdk:"certificate_expiry"`
	DomainsUnderCertificate types.Set    `tfsdk:"domains_under_certificate"`
}

var domainsUnderCertificateAttrType = lo.Assign(
	domainsCommonAttrType,
	map[string]attr.Type{
		"docker_repository_name_override": types.StringType,
	},
)

var domainsUnderCertificateElementType = types.ObjectType{
	AttrTypes: domainsUnderCertificateAttrType,
}

var domainsCommonAttrType = map[string]attr.Type{
	"url":         types.StringType,
	"server_name": types.StringType,
	"type":        types.StringType,
}

func (r *customDomainNameResourceModel) toSubmitAPIModel(_ context.Context, apiModel *customDomainNameSubmitAPIModel) (ds diag.Diagnostics) {
	domainsUnderCertificate := lo.Map(
		r.DomainsUnderCertificate.Elements(),
		func(elem attr.Value, _ int) customDomainNameDomainsCommonAPIModel {
			attr := elem.(types.Object).Attributes()

			return customDomainNameDomainsCommonAPIModel{
				URL:                          attr["url"].(types.String).ValueString(),
				ServerName:                   attr["server_name"].(types.String).ValueString(),
				Type:                         attr["type"].(types.String).ValueString(),
				DockerRepositoryNameOverride: attr["docker_repository_name_override"].(types.String).ValueString(),
			}
		},
	)
	r.DomainsUnderCertificate.Elements()

	*apiModel = customDomainNameSubmitAPIModel{
		customDomainNameCommonAPIModel: customDomainNameCommonAPIModel{
			CertificateName:       r.CertificateName.ValueString(),
			CertificateBody:       r.CertificateBody.ValueString(),
			CertificateChain:      r.CertificateChain.ValueString(),
			CertificatePrivateKey: r.CertificatePrivateKey.ValueString(),
		},
		DomainsUnderCertificate: domainsUnderCertificate,
	}

	return nil
}

func (r *customDomainNameResourceModel) toRenewAPIModel(_ context.Context, apiModel *customDomainNameRenewAPIModel) (ds diag.Diagnostics) {
	domainsUnderCertificate := lo.Map(
		r.DomainsUnderCertificate.Elements(),
		func(elem attr.Value, _ int) customDomainNameDomainsCommonAPIModel {
			attr := elem.(types.Object).Attributes()

			return customDomainNameDomainsCommonAPIModel{
				URL:                          attr["url"].(types.String).ValueString(),
				ServerName:                   attr["server_name"].(types.String).ValueString(),
				Type:                         attr["type"].(types.String).ValueString(),
				DockerRepositoryNameOverride: attr["docker_repository_name_override"].(types.String).ValueString(),
			}
		},
	)
	r.DomainsUnderCertificate.Elements()

	*apiModel = customDomainNameRenewAPIModel{
		CertificateID: r.ID.ValueString(),
		customDomainNameSubmitAPIModel: customDomainNameSubmitAPIModel{
			customDomainNameCommonAPIModel: customDomainNameCommonAPIModel{
				CertificateName:       r.CertificateName.ValueString(),
				CertificateBody:       r.CertificateBody.ValueString(),
				CertificateChain:      r.CertificateChain.ValueString(),
				CertificatePrivateKey: r.CertificatePrivateKey.ValueString(),
			},
			DomainsUnderCertificate: domainsUnderCertificate,
		},
	}

	return nil
}

func (r *customDomainNameResourceModel) fromAPIModel(_ context.Context, apiModel *customDomainNameSSLCertificateAPIModel) (ds diag.Diagnostics) {
	r.ID = types.StringValue(apiModel.CertificateID)
	r.CertificateName = types.StringValue(apiModel.CertificateName)
	r.CertificateBody = types.StringValue(apiModel.CertificateBody)

	certificateChain := types.StringNull()
	if apiModel.CertificateChain != "" {
		certificateChain = types.StringValue(apiModel.CertificateChain)
	}
	r.CertificateChain = certificateChain
	r.CertificateExpiry = types.Int64Value(apiModel.Expiry)
	r.CertificateStatus = types.StringValue(apiModel.CertificateStatus)

	domainsUnderCertificateSet := lo.Map(
		apiModel.DomainsInUse,
		func(domain customDomainNameDomainsCommonAPIModel, _ int) attr.Value {
			dockerRepositoryNameOverride := types.StringNull()
			if domain.DockerRepositoryNameOverride != "" {
				dockerRepositoryNameOverride = types.StringValue(domain.DockerRepositoryNameOverride)
			}

			domainsUnderCertificate, d := types.ObjectValue(
				domainsUnderCertificateAttrType,
				map[string]attr.Value{
					"url":                             types.StringValue(domain.URL),
					"server_name":                     types.StringValue(domain.ServerName),
					"type":                            types.StringValue(domain.Type),
					"docker_repository_name_override": dockerRepositoryNameOverride,
				},
			)

			if d.HasError() {
				ds.Append(d...)
			}

			return domainsUnderCertificate
		},
	)

	domainsUnderCertificate, d := types.SetValue(
		domainsUnderCertificateElementType,
		domainsUnderCertificateSet,
	)

	if d.HasError() {
		ds.Append(d...)
	}

	r.DomainsUnderCertificate = domainsUnderCertificate

	return
}

type customDomainNameCommonAPIModel struct {
	CertificateName       string `json:"certificate_name"`
	CertificateBody       string `json:"certificate_body"`
	CertificateChain      string `json:"certificate_chain"`
	CertificatePrivateKey string `json:"certificate_private_key"`
}

type customDomainNameSubmitAPIModel struct {
	customDomainNameCommonAPIModel
	DomainsUnderCertificate []customDomainNameDomainsCommonAPIModel `json:"domains_under_certificate"`
}

type customDomainNameGetAPIModel struct {
	SSLCertificates []customDomainNameSSLCertificateAPIModel `json:"ssl_certificates"`
}

type customDomainNameSSLCertificateAPIModel struct {
	customDomainNameCommonAPIModel
	Expiry            int64                                   `json:"expiry"`
	CertificateID     string                                  `json:"certificate_id"`
	CertificateStatus string                                  `json:"certificate_status"`
	DomainsInUse      []customDomainNameDomainsCommonAPIModel `json:"domains_in_use"`
}

type customDomainNameDomainsCommonAPIModel struct {
	URL                          string `json:"url"`
	ServerName                   string `json:"server_name"`
	Type                         string `json:"type"`
	DockerRepositoryNameOverride string `json:"docker_repository_name_override,omitempty"`
}

type customDomainNameRenewAPIModel struct {
	customDomainNameSubmitAPIModel
	CertificateID string `json:"certificate_id"`
}

var retryCondition = func(r *resty.Response, _ error) bool {
	return r.StatusCode() == http.StatusConflict
}

func (r *customDomainNameResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan customDomainNameResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var customDomainName customDomainNameSubmitAPIModel
	resp.Diagnostics.Append(plan.toSubmitAPIModel(ctx, &customDomainName)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var errorResp MyJFrogResponseAPIModel
	response, err := r.Client.R().
		SetBody(&customDomainName).
		SetError(&errorResp).
		AddRetryCondition(retryCondition).
		Post("api/jmis/v1/ssl/submit")

	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToCreateResourceError(resp, errorResp.Error())
		return
	}

	var result customDomainNameGetAPIModel
	response, err = r.Client.R().
		SetResult(&result).
		SetError(&errorResp).
		AddRetryCondition(retryCondition).
		Get("api/jmis/v1/ssl")

	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToCreateResourceError(resp, errorResp.Error())
		return
	}

	cert, found := lo.Find(
		result.SSLCertificates,
		func(item customDomainNameSSLCertificateAPIModel) bool {
			return item.customDomainNameCommonAPIModel.CertificateName == plan.CertificateName.ValueString()
		},
	)

	if !found {
		utilfw.UnableToCreateResourceError(resp, fmt.Sprintf("failed to find certificate name %s", plan.CertificateName.ValueString()))
		return
	}

	// Convert from the API data model to the Terraform data model
	// and refresh any attribute values.
	resp.Diagnostics.Append(plan.fromAPIModel(ctx, &cert)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *customDomainNameResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state customDomainNameResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var customDomainName customDomainNameGetAPIModel
	var errorResp MyJFrogResponseAPIModel
	response, err := r.Client.R().
		SetResult(&customDomainName).
		SetError(&errorResp).
		AddRetryCondition(retryCondition).
		Get("api/jmis/v1/ssl")

	if err != nil {
		utilfw.UnableToRefreshResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToRefreshResourceError(resp, errorResp.Error())
		return
	}

	cert, found := lo.Find(
		customDomainName.SSLCertificates,
		func(item customDomainNameSSLCertificateAPIModel) bool {
			return item.CertificateID == state.ID.ValueString()
		},
	)

	if !found {
		utilfw.UnableToRefreshResourceError(resp, fmt.Sprintf("failed to find certificate ID %s", state.ID.ValueString()))
		return
	}

	// Convert from the API data model to the Terraform data model
	// and refresh any attribute values.
	resp.Diagnostics.Append(state.fromAPIModel(ctx, &cert)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *customDomainNameResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan customDomainNameResourceModel
	var state customDomainNameResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the ID from state to plan
	plan.ID = state.ID

	var customDomainName customDomainNameRenewAPIModel
	resp.Diagnostics.Append(plan.toRenewAPIModel(ctx, &customDomainName)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var errorResp MyJFrogResponseAPIModel
	response, err := r.Client.R().
		SetBody(&customDomainName).
		SetError(&errorResp).
		AddRetryCondition(retryCondition).
		Post("api/jmis/v1/ssl/renew")

	if err != nil {
		utilfw.UnableToUpdateResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToUpdateResourceError(resp, errorResp.Error())
		return
	}

	var result customDomainNameGetAPIModel

	response, err = r.Client.R().
		SetResult(&result).
		SetError(&errorResp).
		Get("api/jmis/v1/ssl")

	if err != nil {
		utilfw.UnableToUpdateResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToUpdateResourceError(resp, errorResp.Error())
		return
	}

	cert, found := lo.Find(
		result.SSLCertificates,
		func(item customDomainNameSSLCertificateAPIModel) bool {
			return item.CertificateID == plan.ID.ValueString()
		},
	)

	if !found {
		utilfw.UnableToUpdateResourceError(resp, fmt.Sprintf("failed to find certificate ID %s", plan.ID.ValueString()))
		return
	}

	// Convert from the API data model to the Terraform data model
	// and refresh any attribute values.
	resp.Diagnostics.Append(plan.fromAPIModel(ctx, &cert)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *customDomainNameResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state customDomainNameResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var errorResp MyJFrogResponseAPIModel
	response, err := r.Client.R().
		SetPathParam("id", state.ID.ValueString()).
		SetError(&errorResp).
		AddRetryCondition(retryCondition).
		Delete("api/jmis/v1/ssl/delete/{id}")

	if err != nil {
		utilfw.UnableToDeleteResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToDeleteResourceError(resp, errorResp.Error())
		return
	}

	// If the logic reaches here, it implicitly succeeded and will remove
	// the resource from state if there are no other errors.
}

func (r *customDomainNameResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
