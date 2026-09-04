package myjfrog

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/client"
	"github.com/jfrog/terraform-provider-shared/util"
)

var Version = "1.0.0"

// needs to be exported so make file can update this
var productId = "terraform-provider-myjfrog/" + Version

var _ provider.Provider = (*MyJFrogProvider)(nil)

type MyJFrogProvider struct {
	Meta util.ProviderMetadata
}

type myJFrogProviderModel struct {
	APIToken types.String `tfsdk:"api_token"`
}

func NewProvider() func() provider.Provider {
	return func() provider.Provider {
		return &MyJFrogProvider{}
	}
}

func (p *MyJFrogProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config myJFrogProviderModel

	// Read configuration data into model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiToken := util.CheckEnvVars([]string{"JFROG_MYJFROG_API_TOKEN"}, "")
	if config.APIToken.ValueString() != "" {
		apiToken = config.APIToken.ValueString()
	}

	if apiToken == "" {
		resp.Diagnostics.AddWarning(
			"Missing MyJFrog API Token",
			"MyJFrog API Token was not found in the JFROG_MYJFROG_API_TOKEN environment variable or provider configuration block api_token attribute. Provider will not function.",
		)
	}

	myJFrogClient, err := client.Build("https://my.jfrog.com", productId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Resty client for MyJFrog",
			err.Error(),
		)
		return
	}

	myJFrogClient, err = client.AddAuth(myJFrogClient, "", apiToken)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error adding Auth to Resty client for MyJFrog",
			err.Error(),
		)
		return
	}

	meta := util.ProviderMetadata{
		Client:    myJFrogClient,
		ProductId: productId,
	}

	p.Meta = meta

	resp.DataSourceData = meta
	resp.ResourceData = meta
}

func (p *MyJFrogProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "myjfrog"
	resp.Version = Version
}

func (p *MyJFrogProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewCertificatesDataSource,
	}
}

func (p *MyJFrogProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewIPAllowListResource,
		NewCustomDomainNameResource,
	}
}

func (p *MyJFrogProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				MarkdownDescription: "API token that allows you to make changes to your MyJFrog account. See [Generate a Token in MyJFrog](https://jfrog.com/help/r/jfrog-hosting-models-documentation/generate-a-token-in-myjfrog) for more details. This can also be sourced from the `JFROG_MYJFROG_API_TOKEN` environment variable.",
			},
		},
		MarkdownDescription: "The [JFrog](https://jfrog.com/) MyJFrog provider is used to interact with the features from [MyJFrog portal](https://my.jfrog.com/). The provider needs to be configured with the proper credentials before it can be used.",
	}
}
