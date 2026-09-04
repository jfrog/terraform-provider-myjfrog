package myjfrog

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
	"github.com/samber/lo"
)

var _ datasource.DataSource = (*privateLinksDataSource)(nil)

const privateLinksEndpoint = "/api/jmis/v1/private-link/servers/{serverName}"

type privateLinksDataSource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

func NewPrivateLinksDataSource() datasource.DataSource {
	return &privateLinksDataSource{
		TypeName: "myjfrog_private_links",
	}
}

func (d *privateLinksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = d.TypeName
}

func (d *privateLinksDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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
}

func (d *privateLinksDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"server_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "JFrog server name to search PrivateLink connections for.",
			},
			"private_links": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"private_link_id": schema.StringAttribute{
							Computed: true,
						},
						"status": schema.StringAttribute{
							Computed: true,
						},
					},
				},
				MarkdownDescription: "List of PrivateLink connections for the server and their status values.",
			},
		},
		MarkdownDescription: "Provides a MyJFrog [PrivateLink](https://jfrog.com/help/r/jfrog-hosting-models-documentation/configure-privatelink-for-jfrog-cloud) data source to list PrivateLink connections for a JFrog cloud server.",
	}
}

type privateLinksDataSourceModel struct {
	ServerName   types.String `tfsdk:"server_name"`
	PrivateLinks types.List   `tfsdk:"private_links"`
}

type privateLinksResponseAPIModel struct {
	ServerName   string                       `json:"serverName"`
	PrivateLinks []privateLinkSummaryAPIModel `json:"private_links"`
}

type privateLinkSummaryAPIModel struct {
	PrivateLinkID string `json:"privateLinkId"`
	Status        string `json:"status"`
}

var privateLinkSummaryAttrType = map[string]attr.Type{
	"private_link_id": types.StringType,
	"status":          types.StringType,
}

var privateLinkSummaryElementType = types.ObjectType{AttrTypes: privateLinkSummaryAttrType}

func (d *privateLinksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data privateLinksDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result privateLinksResponseAPIModel
	var apiErr MyJFrogResponseAPIModel
	response, err := d.ProviderData.Client.R().
		SetPathParam("serverName", data.ServerName.ValueString()).
		SetResult(&result).
		SetError(&apiErr).
		Get(privateLinksEndpoint)

	if err != nil {
		resp.Diagnostics.AddError("Unable to read data source", err.Error())
		return
	}

	if response.IsError() {
		resp.Diagnostics.AddError("Unable to read data source", apiErr.Error())
		return
	}

	privateLinks := lo.Map(result.PrivateLinks, func(link privateLinkSummaryAPIModel, _ int) attr.Value {
		obj, ds := types.ObjectValue(
			privateLinkSummaryAttrType,
			map[string]attr.Value{
				"private_link_id": types.StringValue(link.PrivateLinkID),
				"status":          types.StringValue(link.Status),
			},
		)
		if ds.HasError() {
			resp.Diagnostics.Append(ds...)
		}
		return obj
	})

	privateLinksList, ds := types.ListValue(privateLinkSummaryElementType, privateLinks)
	if ds.HasError() {
		resp.Diagnostics.Append(ds...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	data.ServerName = types.StringValue(result.ServerName)
	data.PrivateLinks = privateLinksList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
