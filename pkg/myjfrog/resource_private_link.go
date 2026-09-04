package myjfrog

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
	"github.com/samber/lo"
)

const (
	privateLinkEndpoint       = "/api/jmis/v1/private-link/{privateLinkId}"
	privateLinkAddEndpoint    = "/api/jmis/v1/private-link/add"
	privateLinkDeleteEndpoint = "/api/jmis/v1/private-link/delete"
)

var _ resource.Resource = (*privateLinkResource)(nil)

type privateLinkResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

func NewPrivateLinkResource() resource.Resource {
	return &privateLinkResource{
		TypeName: "myjfrog_private_link",
	}
}

func (r *privateLinkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *privateLinkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
}

func (r *privateLinkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"private_link_id": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(255),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				MarkdownDescription: "Unique identifier for the PrivateLink connection. This value is used as the Terraform resource ID.",
			},
			"server_names": schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.ValueStringsAre(
						stringvalidator.LengthAtLeast(1),
					),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				MarkdownDescription: "List of JFrog server names to associate with the PrivateLink connection. If your JFrog URL is `myserver.jfrog.io`, the server name is `myserver`. " +
					"Changing this value forces replacement of the resource, as the API does not support in-place updates.",
			},
			"servers": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"server_name": schema.StringAttribute{
							Computed: true,
						},
						"status": schema.StringAttribute{
							Computed: true,
						},
					},
				},
				MarkdownDescription: "List of servers and their PrivateLink status values.",
			},
		},
		MarkdownDescription: "Provides a MyJFrog [PrivateLink](https://jfrog.com/help/r/jfrog-hosting-models-documentation/configure-privatelink-for-jfrog-cloud) resource to manage PrivateLink connections for JFrog cloud instances.",
	}
}

type privateLinkResourceModel struct {
	PrivateLinkID types.String `tfsdk:"private_link_id"`
	ServerNames   types.List   `tfsdk:"server_names"`
	Servers       types.List   `tfsdk:"servers"`
}

var privateLinkServerAttrType = map[string]attr.Type{
	"server_name": types.StringType,
	"status":      types.StringType,
}

var privateLinkServerElementType = types.ObjectType{AttrTypes: privateLinkServerAttrType}

func (r *privateLinkResourceModel) toAddAPIModel(_ context.Context) privateLinkAddRequestAPIModel {
	serverNames := []string{}
	_ = r.ServerNames.ElementsAs(context.Background(), &serverNames, false)

	return privateLinkAddRequestAPIModel{
		PrivateLinkID: r.PrivateLinkID.ValueString(),
		ServerNames:   serverNames,
	}
}

func (r *privateLinkResourceModel) fromAPIModel(ctx context.Context, apiModel *privateLinkGetResponseAPIModel) (ds diag.Diagnostics) {
	servers := lo.Map(apiModel.Servers, func(server privateLinkServerAPIModel, _ int) attr.Value {
		obj, d := types.ObjectValue(
			privateLinkServerAttrType,
			map[string]attr.Value{
				"server_name": types.StringValue(server.ServerName),
				"status":      types.StringValue(server.PrivateLinkStatus),
			},
		)
		if d.HasError() {
			ds.Append(d...)
		}
		return obj
	})

	serversList, d := types.ListValue(privateLinkServerElementType, servers)
	if d.HasError() {
		ds.Append(d...)
	}

	r.PrivateLinkID = types.StringValue(apiModel.PrivateLinkID)
	r.Servers = serversList

	return
}

type privateLinkServerAPIModel struct {
	ServerName        string `json:"serverName"`
	PrivateLinkStatus string `json:"privateLinkStatus"`
}

type privateLinkAddRequestAPIModel struct {
	PrivateLinkID string   `json:"privateLinkId"`
	ServerNames   []string `json:"serverNames"`
}

type privateLinkAddResponseAPIModel struct {
	PrivateLinkID string                      `json:"privateLinkId"`
	Results       []privateLinkServerAPIModel `json:"results"`
}

type privateLinkGetResponseAPIModel struct {
	PrivateLinkID string                      `json:"privateLinkId"`
	Servers       []privateLinkServerAPIModel `json:"servers"`
}

func (r *privateLinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan privateLinkResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var serverNames []string
	resp.Diagnostics.Append(plan.ServerNames.ElementsAs(ctx, &serverNames, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiModel := privateLinkAddRequestAPIModel{
		PrivateLinkID: plan.PrivateLinkID.ValueString(),
		ServerNames:   serverNames,
	}

	var result privateLinkAddResponseAPIModel
	var apiErr MyJFrogResponseAPIModel
	response, err := r.ProviderData.Client.R().
		SetBody(&apiModel).
		SetResult(&result).
		SetError(&apiErr).
		Post(privateLinkAddEndpoint)

	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToCreateResourceError(resp, apiErr.Error())
		return
	}

	getResp, err := r.get(ctx, plan.PrivateLinkID.ValueString())
	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	resp.Diagnostics.Append(plan.fromAPIModel(ctx, getResp)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *privateLinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state privateLinkResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	getResp, err := r.get(ctx, state.PrivateLinkID.ValueString())
	if err != nil {
		utilfw.UnableToRefreshResourceError(resp, err.Error())
		return
	}

	resp.Diagnostics.Append(state.fromAPIModel(ctx, getResp)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is never called: both private_link_id and server_names have RequiresReplace,
// so any change triggers replacement. The API has no update endpoint.
func (r *privateLinkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Unexpected Update call",
		"The myjfrog_private_link resource does not support in-place updates. Any change to private_link_id or server_names forces replacement.",
	)
}

func (r *privateLinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state privateLinkResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var serverNames []string
	resp.Diagnostics.Append(state.ServerNames.ElementsAs(ctx, &serverNames, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiModel := privateLinkAddRequestAPIModel{
		PrivateLinkID: state.PrivateLinkID.ValueString(),
		ServerNames:   serverNames,
	}

	var apiErr MyJFrogResponseAPIModel
	response, err := r.ProviderData.Client.R().
		SetBody(&apiModel).
		SetError(&apiErr).
		Delete(privateLinkDeleteEndpoint)

	if err != nil {
		utilfw.UnableToDeleteResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToDeleteResourceError(resp, apiErr.Error())
		return
	}

	// If the logic reaches here, it implicitly succeeded and will remove
	// the resource from state if there are no other errors.
}

func (r *privateLinkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("private_link_id"), req, resp)
}

func (r *privateLinkResource) get(ctx context.Context, privateLinkID string) (*privateLinkGetResponseAPIModel, error) {
	tflog.Info(ctx, fmt.Sprintf("reading private link %s", privateLinkID))

	var result privateLinkGetResponseAPIModel
	var apiErr MyJFrogResponseAPIModel
	response, err := r.ProviderData.Client.R().
		SetPathParam("privateLinkId", privateLinkID).
		SetResult(&result).
		SetError(&apiErr).
		Get(privateLinkEndpoint)

	if err != nil {
		return nil, err
	}

	if response.IsError() {
		return nil, fmt.Errorf("%s", apiErr.Error())
	}

	return &result, nil
}
