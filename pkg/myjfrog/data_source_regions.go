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

const regionsEndpoint = "/api/jmis/v1/regions"

var _ datasource.DataSource = (*regionsDataSource)(nil)

type regionsDataSource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

func NewRegionsDataSource() datasource.DataSource {
	return &regionsDataSource{
		TypeName: "myjfrog_regions",
	}
}

func (d *regionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = d.TypeName
}

func (d *regionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	d.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (d *regionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"regions": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"cloud": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Cloud provider.",
						},
						"region_name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Region name.",
						},
						"region_code": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Region code.",
						},
					},
				},
				MarkdownDescription: "List of regions.",
			},
		},
		MarkdownDescription: "Provides a list of MyJFrog [regions](https://jfrog.com/help/r/jfrog-rest-apis/regions). This data source does not require authentication.",
	}
}

type regionsDataSourceModel struct {
	Regions types.List `tfsdk:"regions"`
}

type regionsAPIResponseModel struct {
	Regions []regionAPIModel `json:"regions"`
}

type regionAPIModel struct {
	Cloud      string `json:"cloud"`
	RegionName string `json:"region_name"`
	RegionCode string `json:"region_code"`
}

type regionModel struct {
	Cloud      types.String `tfsdk:"cloud"`
	RegionName types.String `tfsdk:"region_name"`
	RegionCode types.String `tfsdk:"region_code"`
}

var regionAttrTypes = map[string]attr.Type{
	"cloud":       types.StringType,
	"region_name": types.StringType,
	"region_code": types.StringType,
}

func (d *regionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var result regionsAPIResponseModel

	response, err := d.ProviderData.Client.R().
		SetResult(&result).
		Get(regionsEndpoint)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read regions",
			err.Error(),
		)
		return
	}

	if response.IsError() {
		resp.Diagnostics.AddError(
			"Unable to read regions",
			fmt.Sprintf("%s", response.String()),
		)
		return
	}

	regions := make([]regionModel, 0, len(result.Regions))
	for _, r := range result.Regions {
		regions = append(regions, regionModel{
			Cloud:      types.StringValue(r.Cloud),
			RegionName: types.StringValue(r.RegionName),
			RegionCode: types.StringValue(r.RegionCode),
		})
	}

	var data regionsDataSourceModel
	data.Regions, _ = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: regionAttrTypes}, regions)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
