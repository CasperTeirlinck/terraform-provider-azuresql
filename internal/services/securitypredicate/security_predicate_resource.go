package securitypredicate

import (
	"context"
	"fmt"

	"terraform-provider-azuresql/internal/logging"
	"terraform-provider-azuresql/internal/sql"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &SecurityPredicateResource{}
	_ resource.ResourceWithConfigure   = &SecurityPredicateResource{}
	_ resource.ResourceWithImportState = &SecurityPredicateResource{}
)

func NewSecurityPredicateResource() resource.Resource {
	return &SecurityPredicateResource{}
}

type SecurityPredicateResource struct {
	ConnectionCache *sql.ConnectionCache
}

func (r *SecurityPredicateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_security_predicate"
}

func (r *SecurityPredicateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "SQL database or server user.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for terraform used to import the resource.",
			},
			"database": schema.StringAttribute{
				Required:    true,
				Description: "Id of the database where the user should be created. database or server should be specified.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"security_policy": schema.StringAttribute{
				Required:    true,
				Description: "Terraform resource id of the security policy",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"table": schema.StringAttribute{
				Required:    true,
				Description: "Terraform resource id of the table to which the security predicate applies.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"predicate_id": schema.Int64Attribute{
				Computed:    true,
				Description: "Predicate id of the security predicate.",
			},
			"rule": schema.StringAttribute{
				Required:    true,
				Description: "Security predicate rule.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "Security predicate type.",
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"block", "filter"}...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"block_restriction": schema.StringAttribute{
				Optional:    true,
				Description: "Security predicate type.",
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"", "after insert", "after update", "before update", "before delete"}...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *SecurityPredicateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	ctx = logging.WithDiagnostics(ctx, &resp.Diagnostics)

	var plan SecurityPredicateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	table := plan.Table.ValueString()
	policy := plan.SecurityPolicy.ValueString()
	database := plan.Database.ValueString()
	rule := plan.Rule.ValueString()
	predicateType := plan.Type.ValueString()
	blockRestriction := plan.BlockRestriction.ValueString()
	connection := r.ConnectionCache.Connect(ctx, database, false)

	if logging.HasError(ctx) {
		return
	}

	securityPredicate := sql.CreateSecurityPredicate(ctx, connection, policy, table, predicateType, rule, blockRestriction)

	if logging.HasError(ctx) {
		return
	}

	plan.Id = types.StringValue(securityPredicate.Id)
	plan.PredicateId = types.Int64Value(securityPredicate.PredicateId)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *SecurityPredicateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	ctx = logging.WithDiagnostics(ctx, &resp.Diagnostics)

	var state SecurityPredicateResourceModel

	// Read input configured in data block
	resp.Diagnostics.Append(
		req.State.Get(ctx, &state)...,
	)

	database := state.Database.ValueString()
	connection := r.ConnectionCache.Connect(ctx, database, false)

	if logging.HasError(ctx) {
		return
	}

	securityPredicate := sql.GetSecurityPredicateFromId(ctx, connection, state.Id.ValueString(), false)

	if logging.HasError(ctx) || securityPredicate.Id == "" {
		return
	}

	state.PredicateId = types.Int64Value(securityPredicate.PredicateId)
	state.Rule = types.StringValue(securityPredicate.Rule)
	state.SecurityPolicy = types.StringValue(securityPredicate.SecurityPolicy)
	state.Table = types.StringValue(securityPredicate.Table)
	state.Type = types.StringValue(securityPredicate.PredicateType)
	state.BlockRestriction = types.StringValue(securityPredicate.BlockRestriction)
	state.Id = types.StringValue(securityPredicate.Id)

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *SecurityPredicateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	if req.ProviderData == nil {
		return
	}

	cache, ok := req.ProviderData.(*sql.ConnectionCache)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *sql.Server, got: %T.", req.ProviderData),
		)

		return
	}

	r.ConnectionCache = cache
}

func (r *SecurityPredicateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

}

func (r *SecurityPredicateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	ctx = logging.WithDiagnostics(ctx, &resp.Diagnostics)

	var state SecurityPredicateResourceModel

	// Read input configured in data block
	resp.Diagnostics.Append(
		req.State.Get(ctx, &state)...,
	)

	database := state.Database.ValueString()
	connection := r.ConnectionCache.Connect(ctx, database, false)

	if logging.HasError(ctx) {
		return
	}

	sql.DropSecurityPredicate(ctx, connection, state.Id.ValueString())

	if logging.HasError(ctx) {
		resp.Diagnostics.AddError("Dropping securityPredicate failed", fmt.Sprintf("Dropping securityPredicate %d failed", state.PredicateId.ValueInt64()))
	}
}

func (r *SecurityPredicateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	/*ctx = utils.WithDiagnostics(ctx, &resp.Diagnostics)

	user := sql._user_parse_id(ctx, req.ID)

	if utils.HasError(ctx) {
		return
	}

	resp.State.SetAttribute(ctx, path.Root("connection_string"), user.ConnectionString)
	resp.State.SetAttribute(ctx, path.Root("principal_id"), user.PrincipalId)*/
}
