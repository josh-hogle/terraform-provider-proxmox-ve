// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	proxmox "github.com/luthermonson/go-proxmox"
)

// Ensure proxmoxveProvider satisfies various provider interfaces.
var (
	_ provider.Provider              = &proxmoxveProvider{}
	_ provider.ProviderWithFunctions = &proxmoxveProvider{}
)

// proxmoxveProvider defines the provider implementation.
type proxmoxveProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

type proxmoxveProviderData struct {
	client   *proxmox.Client
	endpoint string
	provider *proxmoxveProvider
}

func (p *proxmoxveProviderData) AddLogContext(ctx context.Context) context.Context {
	ctx = tflog.SetField(ctx, "endpoint", p.endpoint)
	return ctx
}

// proxmoxveProviderModel describes the provider data model.
type proxmoxveProviderModel struct {
	APITokenID                    types.String `tfsdk:"api_token_id"`
	APITokenSecret                types.String `tfsdk:"api_token_secret"`
	APITokenUsername              types.String `tfsdk:"api_token_username"`
	Endpoint                      types.String `tfsdk:"endpoint"`
	IgnoreUntrustedSSLCertificate types.Bool   `tfsdk:"ignore_untrusted_ssl_certificate"`
}

func (p *proxmoxveProvider) Metadata(ctx context.Context, req provider.MetadataRequest,
	resp *provider.MetadataResponse) {

	resp.TypeName = "proxmoxve"
	resp.Version = p.version
}

func (p *proxmoxveProvider) Schema(ctx context.Context, req provider.SchemaRequest,
	resp *provider.SchemaResponse) {

	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_token_id": schema.StringAttribute{
				Description:         "Proxmox VE user API token ID",
				MarkdownDescription: "Proxmox VE user API token ID",
				Required:            true,
				Sensitive:           true,
				//Validators:          []validator.String{},
			},
			"api_token_secret": schema.StringAttribute{
				Description:         "Proxmox VE user API token secret",
				MarkdownDescription: "Proxmox VE user API token secret",
				Required:            true,
				Sensitive:           true,
				//Validators:          []validator.String{},
			},
			"api_token_username": schema.StringAttribute{
				Description:         "Proxmox VE user API token username",
				MarkdownDescription: "Proxmox VE user API token usrename",
				Required:            true,
				Sensitive:           true,
				//Validators:          []validator.String{},
			},
			"endpoint": schema.StringAttribute{
				Description:         "Proxmox VE base URL endpoint (eg: https://server:port)",
				MarkdownDescription: "Proxmox VE base URL endpoint (eg: https://server:port)",
				Required:            true,
				Sensitive:           true,
				//Validators:          []validator.String{},
			},
			"ignore_untrusted_ssl_certificate": schema.BoolAttribute{
				Description:         "Ignore any untrusted / self-signed certificate from the Proxmox VE endpoint",
				MarkdownDescription: "Ignore any untrusted / self-signed certificate from the Proxmox VE endpoint",
				Optional:            true,
			},
		},
	}
}

func (p *proxmoxveProvider) Configure(ctx context.Context, req provider.ConfigureRequest,
	resp *provider.ConfigureResponse) {

	// retrieve provider data from configuration
	var config proxmoxveProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// if the caller provided a configuration value for any of the attributes, it must be a known value
	if config.APITokenID.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token_id"),
			"Unknown Proxmox VE API Token ID",
			"The provider cannot create the Proxmox VE API client as there is an unknown configuration value for "+
				"the API token ID. Either target apply the source of the value first, set the value "+
				"statically in the configuration, or use a variable in the configuration.",
		)
	}
	if config.APITokenSecret.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token_secret"),
			"Unknown Proxmox VE API Token Secret",
			"The provider cannot create the Proxmox VE API client as there is an unknown configuration value for "+
				"the API token secret. Either target apply the source of the value first, set the value "+
				"statically in the configuration, or use a variable in the configuration.",
		)
	}
	if config.APITokenID.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token_username"),
			"Unknown Proxmox VE API Token Username",
			"The provider cannot create the Proxmox VE API client as there is an unknown configuration value for "+
				"the API token username. Either target apply the source of the value first, set the value "+
				"statically in the configuration, or use a variable in the configuration.",
		)
	}
	if config.Endpoint.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Unknown Proxmox VE Endpoint",
			"The provider cannot create the Proxmox VE API client as there is an unknown configuration value for "+
				"the endpoint. Either target apply the source of the value first, set the value "+
				"statically in the configuration, or use a variable in the configuration.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// if any of the configurations are missing, return errors with guidance
	apiTokenID := config.APITokenID.ValueString()
	if apiTokenID == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token_id"),
			"Missing Proxmox VE API Token ID",
			"The provider cannot create the Proxmox VE API client as there is a missing or empty value for "+
				"the API token ID. Either target apply the source of the value first, set the value "+
				"statically in the configuration, or use a variable in the configuration.",
		)
	}
	apiTokenSecret := config.APITokenSecret.ValueString()
	if apiTokenSecret == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token_secret"),
			"Missing Proxmox VE API Token Secret",
			"The provider cannot create the Proxmox VE API client as there is a missing or empty value for "+
				"the API token secret. Either target apply the source of the value first, set the value "+
				"statically in the configuration, or use a variable in the configuration.",
		)
	}
	apiTokenUsername := config.APITokenUsername.ValueString()
	if apiTokenUsername == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token_username"),
			"Missing Proxmox VE API Token Username",
			"The provider cannot create the Proxmox VE API client as there is a missing or empty value for "+
				"the API token username. Either target apply the source of the value first, set the value "+
				"statically in the configuration, or use a variable in the configuration.",
		)
	}
	endpoint := config.Endpoint.ValueString()
	if endpoint == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Missing Proxmox VE Endpoint",
			"The provider cannot create the Proxmox VE API client as there is a missing or empty value for "+
				"the endpoint. Either target apply the source of the value first, set the value "+
				"statically in the configuration, or use a variable in the configuration.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// create the API client
	httpClient := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.IgnoreUntrustedSSLCertificate.ValueBool(),
			},
		},
	}
	client := proxmox.NewClient(
		fmt.Sprintf("%s/api2/json", endpoint),
		proxmox.WithHTTPClient(&httpClient),
		proxmox.WithAPIToken(fmt.Sprintf("%s!%s", apiTokenUsername, apiTokenID), apiTokenSecret))
	version, err := client.Version(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Proxmox VE API: Get Version Failed",
			fmt.Sprintf("Failed to get the Proxmox VE version details from the API:\n\t%s", err.Error()),
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "connected to Proxmox VE server", map[string]any{
		"release":  version.Release,
		"version":  version.Version,
		"repo_id":  version.RepoID,
		"endpoint": endpoint,
	})
	resp.DataSourceData = &proxmoxveProviderData{
		client:   client,
		endpoint: endpoint,
		provider: p,
	}
	resp.ResourceData = resp.DataSourceData
}

func (p *proxmoxveProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}

func (p *proxmoxveProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewVMConfigDataSource,
	}
}

func (p *proxmoxveProvider) Functions(ctx context.Context) []func() function.Function {
	return nil
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &proxmoxveProvider{
			version: version,
		}
	}
}
