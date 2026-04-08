package provider

import (
	webclient "axual-webclient"
	"axual.com/terraform-provider-axual/internal/provider/utils"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &applicationPrincipalResource{}
var _ resource.ResourceWithImportState = &applicationPrincipalResource{}

func NewApplicationPrincipalResource(provider AxualProvider) resource.Resource {
	return &applicationPrincipalResource{
		provider: provider,
	}
}

type applicationPrincipalResource struct {
	provider AxualProvider
}

type applicationPrincipalResourceData struct {
	Principal   types.String `tfsdk:"principal"`
	PrivateKey  types.String `tfsdk:"private_key"`
	Application types.String `tfsdk:"application"`
	Environment types.String `tfsdk:"environment"`
	Custom      types.Bool   `tfsdk:"custom"`
	Id          types.String `tfsdk:"id"`
}

func (r *applicationPrincipalResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_principal"
}

func (r *applicationPrincipalResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "An Application Principal is a security principal (certificate or comparable) that uniquely authenticates an Application in an Environment. Read more: https://docs.axual.io/axual/2025.3/self-service/application-management.html#configuring-application-securityauthentication",

		Attributes: map[string]schema.Attribute{
			"principal": schema.StringAttribute{
				MarkdownDescription: "The principal of an Application for an Environment. Must be PEM-format.",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"private_key": schema.StringAttribute{
				MarkdownDescription: "The private key of a Connector Application for an Environment. Must be PEM-format. If committing terraform configuration(.tf) file in version control repository, please make sure there is a secure way of providing private key for a Connector application's Application Principal. Here are best practices for handling secrets in Terraform: https://blog.gitguardian.com/how-to-handle-secrets-in-terraform/.",
				Optional:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"application": schema.StringAttribute{
				MarkdownDescription: "A valid UID of an existing application",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"environment": schema.StringAttribute{
				MarkdownDescription: "A valid Uid of an existing environment",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Application Principal ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"custom": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "A boolean identifying whether we are creating a custom principal. If true, the custom principal will be stored in `principal` property. Custom principal allows an application with SASL+OAUTHBEARER to produce/consume a topic. Custom Application Principal certificate is used to authenticate your application with an IAM provider using the custom ApplicationPrincipal as Client ID",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *applicationPrincipalResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data applicationPrincipalResourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	applicationPrincipalRequest, err := createApplicationPrincipalRequestFromData(ctx, &data, r)
	if err != nil {
		resp.Diagnostics.AddError("Error creating CREATE request struct for application principal resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	tflog.Info(ctx, fmt.Sprintf("Create application principal request %q", applicationPrincipalRequest))
	applicationPrincipal, err := r.provider.client.CreateApplicationPrincipal(applicationPrincipalRequest)
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for application principal resource", fmt.Sprintf("Error message: %s %s", applicationPrincipal, err))
		return
	}

	var trimmedResponse = strings.Trim(string(applicationPrincipal), "\"")
	returnedUid := strings.ReplaceAll(trimmedResponse, fmt.Sprintf("%s/%s", r.provider.client.ApiURL, "application_principals/"), "")

	data.Id = types.StringValue(returnedUid)

	tflog.Trace(ctx, "Created an application principal resource")
	tflog.Info(ctx, "Saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *applicationPrincipalResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data applicationPrincipalResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	applicationPrincipal, err := r.provider.client.ReadApplicationPrincipal(data.Id.ValueString())
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			tflog.Error(ctx, fmt.Sprintf("Application Principal not found. Id: %s", data.Id.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read application principal, got error: %s", err))
		}
		return
	}

	tflog.Info(ctx, "mapping the resource")
	mapApplicationPrincipalResponseToData(ctx, &data, applicationPrincipal)

	tflog.Info(ctx, "saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *applicationPrincipalResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Client Error", "API does not allow update of application principal. Please delete and recreate the resource.")
}

func (r *applicationPrincipalResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data applicationPrincipalResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Check if the application is a connector
	application, err := r.provider.client.GetApplication(data.Application.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read application, got error: %s", err))
		return
	}

	isConnector := application.ApplicationType == "Connector"
	var deploymentID string
	var shouldRestart bool

	if isConnector {
		applicationURL := fmt.Sprintf("%s/applications/%v", r.provider.client.ApiURL, data.Application.ValueString())
		environmentURL := fmt.Sprintf("%s/environments/%v", r.provider.client.ApiURL, data.Environment.ValueString())

		deploymentResp, err := r.provider.client.FindApplicationDeploymentByApplicationAndEnvironment(applicationURL, environmentURL)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to find deployment, got error: %s", err))
			return
		}
		if len(deploymentResp.Embedded.ApplicationDeploymentResponses) == 0 {
			resp.Diagnostics.AddError("Client Error", "No deployment found for connector")
			return
		}

		deploymentID = deploymentResp.Embedded.ApplicationDeploymentResponses[0].Uid

		// Get deployment status with retry
		var status *webclient.ApplicationDeploymentStatusResponse
		for retries := 0; retries < 3; retries++ {
			status, err = r.provider.client.GetApplicationDeploymentStatus(deploymentID)
			if err == nil {
				break
			}
			if retries < 2 {
				time.Sleep(2 * time.Second)
			}
		}
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get deployment status, got error: %s", err))
			return
		}

		// Stop the connector if needed
		if utils.ShouldStopDeployment("Connector", status) {
			stopRequest := webclient.ApplicationDeploymentOperationRequest{Action: "STOP"}
			err = Retry(3, 10*time.Second, func() error {
				return r.provider.client.OperateApplicationDeployment(deploymentID, "STOP", stopRequest)
			})
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to stop connector after retries, got error: %s", err))
				return
			}

			// Wait for the connector to fully stop (up to 30 seconds)
			stopped := false
			for i := 0; i < 30; i++ {
				time.Sleep(1 * time.Second)
				currentStatus, err := r.provider.client.GetApplicationDeploymentStatus(deploymentID)
				if err != nil {
					tflog.Warn(ctx, fmt.Sprintf("Error polling stop status on attempt %d: %s", i+1, err))
					continue
				}
				if !utils.ShouldStopDeployment("Connector", currentStatus) {
					stopped = true
					break
				}
			}
			if !stopped {
				resp.Diagnostics.AddError("Client Error", "Connector did not stop within 30 seconds")
				return
			}
		}

		principals, err := r.provider.client.FindApplicationPrincipalByApplicationAndEnvironment(
			fmt.Sprintf("%s/applications/%v", r.provider.client.ApiURL, data.Application.ValueString()),
			fmt.Sprintf("%s/environments/%v", r.provider.client.ApiURL, data.Environment.ValueString()),
		)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list application principals, got error: %s", err))
			return
		}
		shouldRestart = len(principals.Embedded.ApplicationPrincipalResponses) > 1
	}

	// Delete the application principal
	err = r.provider.client.DeleteApplicationPrincipal(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete application principal, got error: %s", err))
		return
	}

	// Restart the connector only if at least one principal remains after deletion
	if isConnector && deploymentID != "" && shouldRestart {
		status, err := r.provider.client.GetApplicationDeploymentStatus(deploymentID)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get deployment status after principal deletion, got error: %s", err))
			return
		}

		if utils.ShouldStartDeployment("Connector", status) {
			startRequest := webclient.ApplicationDeploymentOperationRequest{Action: "START"}
			err = Retry(3, 10*time.Second, func() error {
				return r.provider.client.OperateApplicationDeployment(deploymentID, "START", startRequest)
			})
			if err != nil {
				if strings.Contains(err.Error(), "Invalid action for this state of deployment") {
					tflog.Info(ctx, "Connector appears to be already running - START not needed")
				} else {
					resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to start connector after principal deletion, got error: %s", err))
					return
				}
			}
		}
	}
}

func (r *applicationPrincipalResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func createApplicationPrincipalRequestFromData(ctx context.Context, data *applicationPrincipalResourceData, r *applicationPrincipalResource) ([1]webclient.ApplicationPrincipalRequest, error) {
	rawEnvironment, err := data.Environment.ToTerraformValue(ctx)
	if err != nil {
		return [1]webclient.ApplicationPrincipalRequest{}, err
	}
	var environment string
	err = rawEnvironment.As(&environment)
	if err != nil {
		return [1]webclient.ApplicationPrincipalRequest{}, err
	}
	environment = fmt.Sprintf("%s/%v", r.provider.client.ApiURL, environment)

	rawApplication, err := data.Application.ToTerraformValue(ctx)
	if err != nil {
		return [1]webclient.ApplicationPrincipalRequest{}, err
	}
	var application string
	err = rawApplication.As(&application)
	if err != nil {
		return [1]webclient.ApplicationPrincipalRequest{}, err
	}
	application = fmt.Sprintf("%s/applications/%v", r.provider.client.ApiURL, application)

	var applicationPrincipalRequestArray [1]webclient.ApplicationPrincipalRequest
	applicationPrincipalRequestArray[0] =
		webclient.ApplicationPrincipalRequest{
			Principal:   strings.TrimSpace(data.Principal.ValueString()),
			Application: application,
			Environment: environment,
		}
	// optional fields
	if !data.Custom.IsNull() && data.Custom.ValueBool() {
		applicationPrincipalRequestArray[0].Custom = data.Custom.ValueBool()
	}
	if !data.PrivateKey.IsNull() {
		applicationPrincipalRequestArray[0].PrivateKey = strings.TrimSpace(data.PrivateKey.ValueString())
	}
	return applicationPrincipalRequestArray, err
}

func mapApplicationPrincipalResponseToData(_ context.Context, data *applicationPrincipalResourceData, applicationPrincipal *webclient.ApplicationPrincipalResponse) {
	data.Id = types.StringValue(applicationPrincipal.Uid)
	data.Environment = types.StringValue(applicationPrincipal.Embedded.Environment.Uid)
	data.Application = types.StringValue(applicationPrincipal.Embedded.Application.Uid)
	// Branch on API type: only SSL deals with PEM certificate files.
	if applicationPrincipal.Type == "OAUTH" {
		data.Custom = types.BoolValue(true)
		data.Principal = types.StringValue(applicationPrincipal.Principal)
	} else {
		// SSL: applicationPem contains the full PEM certificate chain.
		// Preserve existing state value when only whitespace differs. The API returns
		// trimmed values, but the config (from file()) may include a trailing newline.
		apiPrincipal := applicationPrincipal.ApplicationPem
		if !data.Principal.IsNull() && !data.Principal.IsUnknown() &&
			strings.TrimSpace(data.Principal.ValueString()) == strings.TrimSpace(apiPrincipal) {
			// Keep existing state value — semantically equal
		} else {
			data.Principal = types.StringValue(apiPrincipal)
		}
	}
}
