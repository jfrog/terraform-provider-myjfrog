package myjfrog

import (
	"context"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
	"github.com/samber/lo"
)

var _ datasource.DataSource = (*sslSslCertificatesDataSource)(nil)

type sslSslCertificatesDataSource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
	Client       *resty.Client
}

func NewSslCertificatesDataSource() datasource.DataSource {
	return &sslSslCertificatesDataSource{
		TypeName: "myjfrog_ssl_certificates",
	}
}

func (d *sslSslCertificatesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = d.TypeName
}

func (d *sslSslCertificatesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.ProviderData = req.ProviderData.(util.ProviderMetadata)

	if d.ProviderData.Client == nil {
		resp.Diagnostics.AddError(
			"Client not configured in provider",
			"MyJFrog Resty client is not configured due to missing `api_token` attribute in provider configuration, or missing `JFROG_MYJFROG_API_TOKEN` env var.",
		)
	}

	d.Client = d.ProviderData.Client.Clone().
		SetRetryCount(10).
		SetRetryWaitTime(1 * time.Minute).
		SetRetryMaxWaitTime(2 * time.Minute)
}

func (d *sslSslCertificatesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identifier for the data source.",
			},
			"certificates": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"certificate_id": schema.StringAttribute{
							Computed: true,
						},
						"certificate_name": schema.StringAttribute{
							Computed: true,
						},
						"certificate_body": schema.StringAttribute{
							Computed: true,
						},
						"certificate_chain": schema.StringAttribute{
							Computed: true,
						},
						"certificate_status": schema.StringAttribute{
							Computed: true,
						},
						"certificate_expiry": schema.Int64Attribute{
							Computed: true,
						},
						"domains_in_use": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"url": schema.StringAttribute{
										Computed: true,
									},
									"server_name": schema.StringAttribute{
										Computed: true,
									},
									"type": schema.StringAttribute{
										Computed: true,
									},
									"docker_repository_name_override": schema.StringAttribute{
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
		},
		MarkdownDescription: "Fetches the list of SSL certificates configured in MyJFrog. Also see [Custom Domain Name REST API](https://jfrog.com/help/r/jfrog-rest-apis/custom-domain-name-rest-apis).",
	}
}

type sslSslCertificatesDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Certificates types.List   `tfsdk:"certificates"`
}

type certificatesCertificateModel struct {
	CertificateID     types.String `tfsdk:"certificate_id"`
	CertificateName   types.String `tfsdk:"certificate_name"`
	CertificateBody   types.String `tfsdk:"certificate_body"`
	CertificateChain  types.String `tfsdk:"certificate_chain"`
	CertificateStatus types.String `tfsdk:"certificate_status"`
	CertificateExpiry types.Int64  `tfsdk:"certificate_expiry"`
	DomainsInUse      types.List   `tfsdk:"domains_in_use"`
}

var domainsInUseAttrType = lo.Assign(
	domainsCommonAttrType,
	map[string]attr.Type{
		"docker_repository_name_override": types.StringType,
	},
)

var domainsInUseElementType = types.ObjectType{
	AttrTypes: domainsInUseAttrType,
}

var certificateAttrType = map[string]attr.Type{
	"certificate_id":     types.StringType,
	"certificate_name":   types.StringType,
	"certificate_body":   types.StringType,
	"certificate_chain":  types.StringType,
	"certificate_status": types.StringType,
	"certificate_expiry": types.Int64Type,
	"domains_in_use":     types.ListType{ElemType: domainsInUseElementType},
}

var certificateElementType = types.ObjectType{
	AttrTypes: certificateAttrType,
}

func (d *sslSslCertificatesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	go util.SendUsage(ctx, d.ProviderData.Client.R(), d.ProviderData.ProductId, d.TypeName)

	var state sslSslCertificatesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result customDomainNameGetAPIModel
	var errorResp MyJFrogResponseAPIModel
	response, err := d.Client.R().
		SetResult(&result).
		SetError(&errorResp).
		AddRetryCondition(retryCondition).
		Get("api/jmis/v1/ssl")

	if err != nil {
		resp.Diagnostics.AddError("Unable to Refresh Data Source", err.Error())
		return
	}

	if response.IsError() {
		errMsg := errorResp.Error()
		if response.StatusCode() == http.StatusConflict {
			errMsg = conflictHint(errMsg)
		}
		resp.Diagnostics.AddError("Unable to Refresh Data Source", errMsg)
		return
	}

	certificates, ds := toSslCertificatesList(ctx, result.SSLCertificates)
	resp.Diagnostics.Append(ds...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.ID = types.StringValue("myjfrog_ssl_certificates")
	state.Certificates = certificates

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func toSslCertificatesList(ctx context.Context, apiCertificates []customDomainNameSSLCertificateAPIModel) (types.List, diag.Diagnostics) {
	var ds diag.Diagnostics

	elements := lo.Map(
		apiCertificates,
		func(apiCert customDomainNameSSLCertificateAPIModel, _ int) attr.Value {
			domainsInUse, d := toDomainsInUseList(ctx, apiCert.DomainsInUse)
			ds.Append(d...)

			certificateChain := types.StringNull()
			if apiCert.CertificateChain != "" {
				certificateChain = types.StringValue(apiCert.CertificateChain)
			}

			certificate, d := types.ObjectValue(
				certificateAttrType,
				map[string]attr.Value{
					"certificate_id":     types.StringValue(apiCert.CertificateID),
					"certificate_name":   types.StringValue(apiCert.CertificateName),
					"certificate_body":   types.StringValue(apiCert.CertificateBody),
					"certificate_chain":  certificateChain,
					"certificate_status": types.StringValue(apiCert.CertificateStatus),
					"certificate_expiry": types.Int64Value(apiCert.Expiry),
					"domains_in_use":     domainsInUse,
				},
			)
			ds.Append(d...)

			return certificate
		},
	)

	certificates, d := types.ListValue(
		certificateElementType,
		elements,
	)
	ds.Append(d...)

	return certificates, ds
}

func toDomainsInUseList(ctx context.Context, apiDomains []customDomainNameDomainsCommonAPIModel) (types.List, diag.Diagnostics) {
	var ds diag.Diagnostics

	elements := lo.Map(
		apiDomains,
		func(domain customDomainNameDomainsCommonAPIModel, _ int) attr.Value {
			dockerRepositoryNameOverride := types.StringNull()
			if domain.DockerRepositoryNameOverride != "" {
				dockerRepositoryNameOverride = types.StringValue(domain.DockerRepositoryNameOverride)
			}

			obj, d := types.ObjectValue(
				domainsInUseAttrType,
				map[string]attr.Value{
					"url":                             types.StringValue(domain.URL),
					"server_name":                     types.StringValue(domain.ServerName),
					"type":                            types.StringValue(domain.Type),
					"docker_repository_name_override": dockerRepositoryNameOverride,
				},
			)
			ds.Append(d...)

			return obj
		},
	)

	domainsInUse, d := types.ListValue(
		domainsInUseElementType,
		elements,
	)
	ds.Append(d...)

	return domainsInUse, ds
}
