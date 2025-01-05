package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &vmConfigDataSource{}
	_ datasource.DataSourceWithConfigure = &vmConfigDataSource{}
)

func NewVMConfigDataSource() datasource.DataSource {
	return &vmConfigDataSource{}
}

type vmConfigDataSource struct {
	providerData *proxmoxveProviderData
}

type vmConfigDataSourceModel struct {
	Data   *vmConfigDataSourceDataModel   `tfsdk:"data"`
	Filter *vmConfigDataSourceFilterModel `tfsdk:"filter"`
}

type vmConfigDataSourceFilterModel struct {
	NodeName types.String `tfsdk:"node_name"`
	VMID     types.Int32  `tfsdk:"vm_id"`
}

type vmConfigDataSourceDataModel struct {
	Name              types.String                              `tfsdk:"name"`
	Node              types.String                              `tfsdk:"node"`
	NetworkInterfaces []vmConfigDataSourceNetworkInterfaceModel `tfsdk:"network_interfaces"`
	Status            types.String                              `tfsdk:"status"`
	VMID              types.Int32                               `tfsdk:"vm_id"`
}

type vmConfigDataSourceNetworkInterfaceModel struct {
	Bridge          types.String  `tfsdk:"bridge"`
	Firewall        types.Bool    `tfsdk:"firewall"`
	HardwareAddress types.String  `tfsdk:"mac_addr"`
	LinkDown        types.Bool    `tfsdk:"link_down"`
	Model           types.String  `tfsdk:"model"`
	MTU             types.Int32   `tfsdk:"mtu"`
	Queues          types.Int32   `tfsdk:"queues"`
	Rate            types.Int32   `tfsdk:"rate"`
	RawConfig       types.String  `tfsdk:"raw_config"`
	Tag             types.Int32   `tfsdk:"tag"`
	Trunks          []types.Int32 `tfsdk:"trunks"`
}

func (d *vmConfigDataSource) Configure(_ context.Context, req datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse) {

	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*proxmoxveProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type", fmt.Sprintf(
				"Expected *proxmoxveProviderData, got: %T. Please report this issue to the provider developers.",
				req.ProviderData),
		)
		return
	}

	d.providerData = data
}

func (d *vmConfigDataSource) Metadata(_ context.Context, req datasource.MetadataRequest,
	resp *datasource.MetadataResponse) {

	resp.TypeName = req.ProviderTypeName + "_vm_config"
}

func (d *vmConfigDataSource) Schema(_ context.Context, req datasource.SchemaRequest,
	resp *datasource.SchemaResponse) {

	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"data": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Computed: true,
					},
					"node": schema.StringAttribute{
						Computed: true,
					},
					"network_interfaces": schema.ListNestedAttribute{
						Computed: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"model": schema.StringAttribute{
									Computed: true,
									Optional: true,
								},
								"bridge": schema.StringAttribute{
									Computed: true,
									Optional: true,
								},
								"firewall": schema.BoolAttribute{
									Computed: true,
									Optional: true,
								},
								"link_down": schema.BoolAttribute{
									Computed: true,
									Optional: true,
								},
								"mac_addr": schema.StringAttribute{
									Computed: true,
									Optional: true,
								},
								"mtu": schema.Int32Attribute{
									Computed: true,
									Optional: true,
								},
								"queues": schema.Int32Attribute{
									Computed: true,
									Optional: true,
								},
								"rate": schema.Int32Attribute{
									Computed: true,
									Optional: true,
								},
								"raw_config": schema.StringAttribute{
									Computed: true,
								},
								"tag": schema.Int32Attribute{
									Computed: true,
									Optional: true,
								},
								"trunks": schema.ListAttribute{
									Computed:    true,
									ElementType: types.Int32Type,
									Optional:    true,
								},
							},
						},
					},
					"status": schema.StringAttribute{
						Computed: true,
					},
					"vm_id": schema.Int32Attribute{
						Computed: true,
					},
				},
			},
			"filter": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"node_name": schema.StringAttribute{
						Required: true,
					},
					"vm_id": schema.Int32Attribute{
						Required: true,
					},
				},
			},
		},
	}
}

func (d *vmConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest,
	resp *datasource.ReadResponse) {

	ctx = d.providerData.AddLogContext(ctx)

	// read configuration
	var config vmConfigDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// make sure a VM ID and node are specified
	if config.Filter == nil {
		resp.Diagnostics.AddError(
			"Filter Is Required", "You must specify a filter to retrieve the VM configuration.",
		)
		return
	}
	if config.Filter.NodeName.IsNull() || config.Filter.NodeName.IsUnknown() {
		resp.Diagnostics.AddError(
			"Filter Node Name Is Required", "You must specify a PVE cluster node name to retrieve the VM configuration.",
		)
		return
	}
	nodeName := config.Filter.NodeName.ValueString()
	if config.Filter.VMID.IsNull() || config.Filter.VMID.IsUnknown() {
		resp.Diagnostics.AddError(
			"Filter VM ID Is Required", "You must specify a VM ID to retrieve the VM configuration.",
		)
		return
	}
	vmID := int(config.Filter.VMID.ValueInt32())

	// query for the configuration
	node, err := d.providerData.client.Node(ctx, nodeName)
	if err != nil {
		tflog.Error(ctx, "failed to locate cluster node", map[string]any{
			"node_name": nodeName,
			"error":     err.Error(),
		})
		resp.Diagnostics.AddError(
			"Proxmox VE API: Failed to Locate Node",
			fmt.Sprintf("Failed to locate the cluster node '%s':\n\t%s", nodeName, err.Error()),
		)
		return
	}
	vm, err := node.VirtualMachine(ctx, vmID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Proxmox VE API: Failed to Retrieve VM",
			fmt.Sprintf("Failed to retrieve the virtual machine with the ID '%d':\n\t%s", vmID, err.Error()),
		)
		return
	}
	tflog.Info(ctx, "located VM", map[string]any{"vm": vm})

	// map the response to the model
	state := vmConfigDataSourceModel{
		Data: &vmConfigDataSourceDataModel{
			Name:              types.StringValue(vm.Name),
			NetworkInterfaces: []vmConfigDataSourceNetworkInterfaceModel{},
			Node:              types.StringValue(vm.Node),
			Status:            types.StringValue(vm.Status),
			VMID:              config.Filter.VMID,
		},
		Filter: config.Filter,
	}
	if vm.VirtualMachineConfig != nil {
		for name, config := range vm.VirtualMachineConfig.MergeNets() {
			tflog.Info(ctx, "parsing network interface", map[string]any{"name": name, "config": config, "vm_id": vmID})
			if config == "" {
				continue
			}
			state.Data.NetworkInterfaces = append(state.Data.NetworkInterfaces,
				d.parseNetworkConfig(ctx, config, resp.Diagnostics))
		}
	} else {
		tflog.Warn(ctx, "VM config is nil", map[string]any{"vm_id": vmID})
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (d *vmConfigDataSource) parseNetworkConfig(_ context.Context, config string,
	diag diag.Diagnostics) vmConfigDataSourceNetworkInterfaceModel {

	iface := vmConfigDataSourceNetworkInterfaceModel{
		RawConfig: types.StringValue(config),
	}
	pairs := strings.Split(config, ",")
	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		key := kv[0]
		value := kv[1]

		switch key {
		case "model":
			iface.Model = types.StringValue(value)
		case "bridge":
			iface.Bridge = types.StringValue(value)
		case "firewall":
			val, err := strconv.ParseBool(value)
			if err != nil {
				diag.AddError(
					"Unexpected VM Config Value",
					fmt.Sprintf(
						"The value for the 'firewall' property for the network interface was not expected: %s",
						err.Error()),
				)
				continue
			}
			iface.Firewall = types.BoolValue(val)
		case "link_down":
			val, err := strconv.ParseBool(value)
			if err != nil {
				diag.AddError(
					"Unexpected VM Config Value",
					fmt.Sprintf(
						"The value for the 'link_down' property for the network interface was not expected: %s",
						err.Error()),
				)
				continue
			}
			iface.LinkDown = types.BoolValue(val)
		case "macaddr", "virtio":
			iface.HardwareAddress = types.StringValue(value)
		case "mtu":
			val, err := strconv.ParseInt(value, 10, 32)
			if err != nil {
				diag.AddError(
					"Unexpected VM Config Value",
					fmt.Sprintf(
						"The value for the 'mtu' property for the network interface was not expected: %s",
						err.Error()),
				)
				continue
			}
			iface.MTU = types.Int32Value(int32(val))
		case "queues":
			val, err := strconv.ParseInt(value, 10, 32)
			if err != nil {
				diag.AddError(
					"Unexpected VM Config Value",
					fmt.Sprintf(
						"The value for the 'queues' property for the network interface was not expected: %s",
						err.Error()),
				)
				continue
			}
			iface.Queues = types.Int32Value(int32(val))
		case "rate":
			val, err := strconv.ParseInt(value, 10, 32)
			if err != nil {
				diag.AddError(
					"Unexpected VM Config Value",
					fmt.Sprintf(
						"The value for the 'rate' property for the network interface was not expected: %s",
						err.Error()),
				)
				continue
			}
			iface.Rate = types.Int32Value(int32(val))
		case "tag":
			val, err := strconv.ParseInt(value, 10, 32)
			if err != nil {
				diag.AddError(
					"Unexpected VM Config Value",
					fmt.Sprintf(
						"The value for the 'tag' property for the network interface was not expected: %s",
						err.Error()),
				)
				continue
			}
			iface.Tag = types.Int32Value(int32(val))
		case "trunks":
			iface.Trunks = []types.Int32{}
			for _, trunk := range strings.Split(value, ";") {
				val, err := strconv.ParseInt(trunk, 10, 32)
				if err != nil {
					diag.AddError(
						"Unexpected VM Config Value",
						fmt.Sprintf(
							"The value for the 'trunks' property for the network interface was not expected: %s",
							err.Error()),
					)
					continue
				}
				iface.Trunks = append(iface.Trunks, types.Int32Value(int32(val)))
			}
		}
	}
	return iface
}
