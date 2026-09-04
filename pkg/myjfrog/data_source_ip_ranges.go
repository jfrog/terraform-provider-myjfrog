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

const ipRangesEndpoint = "/api/jmis/v1/ip-ranges"

var _ datasource.DataSource = (*ipRangesDataSource)(nil)

type ipRangesDataSource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

func NewIPRangesDataSource() datasource.DataSource {
	return &ipRangesDataSource{
		TypeName: "myjfrog_ip_ranges",
	}
}

func (d *ipRangesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = d.TypeName
}

func (d *ipRangesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	d.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (d *ipRangesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"ip_ranges": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"cidr": schema.ListAttribute{
							ElementType:         types.StringType,
							Computed:            true,
							MarkdownDescription: "List of CIDR blocks.",
						},
						"region": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Region name.",
						},
						"service": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Service name.",
						},
						"cloud": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Cloud provider.",
						},
					},
				},
				MarkdownDescription: "List of IP ranges.",
			},
		},
		MarkdownDescription: "Provides a list of MyJFrog [IP ranges](https://jfrog.com/help/r/jfrog-rest-apis/ip-ranges). This data source does not require authentication.",
	}
}

type ipRangesDataSourceModel struct {
	IPRanges types.List `tfsdk:"ip_ranges"`
}

type ipRangeAPIModel struct {
	CIDR    []string `json:"cidr"`
	Region  string   `json:"region"`
	Service string   `json:"service"`
	Cloud   string   `json:"cloud"`
}

func (d *ipRangesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var result []ipRangeAPIModel

	response, err := d.ProviderData.Client.R().
		SetResult(&result).
		Get(ipRangesEndpoint)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read IP ranges",
			err.Error(),
		)
		return
	}

	if response.IsError() {
		resp.Diagnostics.AddError(
			"Unable to read IP ranges",
			fmt.Sprintf("%s", response.String()),
		)
		return
	}

	ipRanges := make([]ipRangeModel, 0, len(result))
	for _, r := range result {
		cidrs, diags := types.ListValueFrom(ctx, types.StringType, r.CIDR)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		ipRanges = append(ipRanges, ipRangeModel{
			CIDR:    cidrs,
			Region:  types.StringValue(r.Region),
			Service: types.StringValue(r.Service),
			Cloud:   types.StringValue(r.Cloud),
		})
	}

	var data ipRangesDataSourceModel
	data.IPRanges, _ = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: ipRangeAttrTypes}, ipRanges)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type ipRangeModel struct {
	CIDR    types.List   `tfsdk:"cidr"`
	Region  types.String `tfsdk:"region"`
	Service types.String `tfsdk:"service"`
	Cloud   types.String `tfsdk:"cloud"`
}

var ipRangeAttrTypes = map[string]attr.Type{
	"cidr":    types.ListType{ElemType: types.StringType},
	"region":  types.StringType,
	"service": types.StringType,
	"cloud":   types.StringType,
}
