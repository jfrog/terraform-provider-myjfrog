package myjfrog

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
)

const serversEndpoint = "/api/jmis/v1/servers"

var _ datasource.DataSource = (*serversDataSource)(nil)

type serversDataSource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

func NewServersDataSource() datasource.DataSource {
	return &serversDataSource{
		TypeName: "myjfrog_servers",
	}
}

func (d *serversDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = d.TypeName
}

func (d *serversDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *serversDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"servers": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"server_name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Server name.",
						},
						"server_type": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Server type.",
						},
						"cloud_provider": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Cloud provider.",
						},
						"region": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Region name.",
						},
					},
				},
				MarkdownDescription: "List of servers.",
			},
		},
		MarkdownDescription: "Provides a list of MyJFrog [servers](https://jfrog.com/help/r/jfrog-rest-apis/servers). This data source requires an API token.",
	}
}

type serversDataSourceModel struct {
	Servers types.List `tfsdk:"servers"`
}

type serversAPIResponseModel struct {
	Servers []serverAPIModel `json:"servers"`
}

type serverAPIModel struct {
	ServerName    string `json:"server_name"`
	ServerType    string `json:"server_type"`
	CloudProvider string `json:"cloud_provider"`
	Region        string `json:"region"`
}

type serverModel struct {
	ServerName    types.String `tfsdk:"server_name"`
	ServerType    types.String `tfsdk:"server_type"`
	CloudProvider types.String `tfsdk:"cloud_provider"`
	Region        types.String `tfsdk:"region"`
}

var serverAttrTypes = map[string]attr.Type{
	"server_name":    types.StringType,
	"server_type":    types.StringType,
	"cloud_provider": types.StringType,
	"region":         types.StringType,
}

func (d *serversDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var result serversAPIResponseModel

	response, err := d.ProviderData.Client.R().
		SetResult(&result).
		Get(serversEndpoint)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read servers",
			err.Error(),
		)
		return
	}

	if response.IsError() {
		resp.Diagnostics.AddError(
			"Unable to read servers",
			fmt.Sprintf("%s", response.String()),
		)
		return
	}

	servers := make([]serverModel, 0, len(result.Servers))
	for _, s := range result.Servers {
		servers = append(servers, serverModel{
			ServerName:    types.StringValue(s.ServerName),
			ServerType:    types.StringValue(s.ServerType),
			CloudProvider: types.StringValue(s.CloudProvider),
			Region:        types.StringValue(s.Region),
		})
	}

	var data serversDataSourceModel
	data.Servers, _ = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: serverAttrTypes}, servers)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
