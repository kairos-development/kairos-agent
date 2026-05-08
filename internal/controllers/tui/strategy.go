package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/kairos-development/kairos-agent/internal/strategy/builder"
	"github.com/shopspring/decimal"
)

// strategyBuilderService defines the interface for strategy builder operations.
type strategyBuilderService interface {
	ListTemplates(context.Context) ([]*builder.StrategyTemplate, error)
	GetTemplate(context.Context, string) (*builder.StrategyTemplate, error)
	CreateInstance(context.Context, builder.CreateInstanceInput) (*builder.StrategyInstance, error)
	UpdateInstance(context.Context, string, builder.UpdateInstanceInput) (*builder.StrategyInstance, error)
	GetInstance(context.Context, string) (*builder.StrategyInstance, error)
	ListInstances(context.Context) ([]*builder.StrategyInstance, error)
	DeleteInstance(context.Context, string) error
	EnableInstance(context.Context, string) error
	DisableInstance(context.Context, string) error
}

// handleStrategyCommand handles strategy builder commands.
func (m *Model) handleStrategyCommand(ctx context.Context, args []string) {
	if m.strategyBuilder == nil {
		m.appendLine("strategy builder not available")
		return
	}

	if len(args) == 0 {
		m.appendLine("usage: /strategy <list|templates|create|update|enable|disable|delete|show>")
		return
	}

	subcommand := args[0]
	switch subcommand {
	case "templates":
		m.handleStrategyTemplates(ctx)
	case "list":
		m.handleStrategyList(ctx)
	case "create":
		m.handleStrategyCreate(ctx, args[1:])
	case "show":
		m.handleStrategyShow(ctx, args[1:])
	case "update":
		m.handleStrategyUpdate(ctx, args[1:])
	case "enable":
		m.handleStrategyEnable(ctx, args[1:])
	case "disable":
		m.handleStrategyDisable(ctx, args[1:])
	case "delete":
		m.handleStrategyDelete(ctx, args[1:])
	default:
		m.appendLine("unknown strategy subcommand: " + subcommand)
	}
}

// handleStrategyTemplates lists available strategy templates.
func (m *Model) handleStrategyTemplates(ctx context.Context) {
	templates, err := m.strategyBuilder.ListTemplates(ctx)
	if err != nil {
		m.appendLine("failed to list templates: " + err.Error())
		return
	}

	if len(templates) == 0 {
		m.appendLine("no templates available")
		return
	}

	m.appendLine(fmt.Sprintf("available templates (%d):", len(templates)))
	for _, tmpl := range templates {
		m.appendLine(fmt.Sprintf("  %s: %s (%s)", tmpl.ID, tmpl.Name, tmpl.Category))
	}
}

// handleStrategyList lists all strategy instances.
func (m *Model) handleStrategyList(ctx context.Context) {
	instances, err := m.strategyBuilder.ListInstances(ctx)
	if err != nil {
		m.appendLine("failed to list strategies: " + err.Error())
		return
	}

	if len(instances) == 0 {
		m.appendLine("no strategies configured")
		return
	}

	m.appendLine(fmt.Sprintf("configured strategies (%d):", len(instances)))
	for _, inst := range instances {
		status := "disabled"
		if inst.Enabled {
			status = "enabled"
		}
		idDisplay := inst.ID
		if len(idDisplay) > 8 {
			idDisplay = idDisplay[:8]
		}
		m.appendLine(fmt.Sprintf("  [%s] %s: %s on %s (%s)", idDisplay, inst.Name, inst.TemplateID, inst.Symbol, status))
	}
}

// handleStrategyCreate creates a new strategy instance.
func (m *Model) handleStrategyCreate(ctx context.Context, args []string) {
	if len(args) < 3 {
		m.appendLine("usage: /strategy create <template_id> <name> <symbol> [param=value...]")
		return
	}

	templateID := args[0]
	name := args[1]
	symbol := args[2]

	// Get template to validate parameters
	template, err := m.strategyBuilder.GetTemplate(ctx, templateID)
	if err != nil {
		m.appendLine("failed to get template: " + err.Error())
		return
	}

	// Parse parameters
	params := make(map[string]interface{})
	for i := 3; i < len(args); i++ {
		parts := strings.SplitN(args[i], "=", 2)
		if len(parts) != 2 {
			m.appendLine("invalid parameter format: " + args[i])
			return
		}

		key := parts[0]
		value := parts[1]

		// Find parameter definition
		var paramDef *builder.ParameterDefinition
		for _, def := range template.Parameters {
			if def.Key == key {
				paramDef = &def
				break
			}
		}

		if paramDef == nil {
			m.appendLine("unknown parameter: " + key)
			return
		}

		// Parse value based on type
		parsedValue, err := m.parseParameterValue(paramDef.Type, value)
		if err != nil {
			m.appendLine(fmt.Sprintf("invalid value for %s: %s", key, err.Error()))
			return
		}

		params[key] = parsedValue
	}

	// Create instance
	input := builder.CreateInstanceInput{
		TemplateID:  templateID,
		Name:        name,
		Description: fmt.Sprintf("Created via TUI at %s", ctx.Value("timestamp")),
		Symbol:      symbol,
		Parameters:  params,
	}

	instance, err := m.strategyBuilder.CreateInstance(ctx, input)
	if err != nil {
		m.appendLine("failed to create strategy: " + err.Error())
		return
	}

	idDisplay := instance.ID
	if len(idDisplay) > 8 {
		idDisplay = idDisplay[:8]
	}
	m.appendLine(fmt.Sprintf("strategy created: %s (%s)", instance.Name, idDisplay))
}

// handleStrategyShow displays details of a strategy instance.
func (m *Model) handleStrategyShow(ctx context.Context, args []string) {
	if len(args) == 0 {
		m.appendLine("usage: /strategy show <instance_id>")
		return
	}

	instanceID := args[0]
	instance, err := m.strategyBuilder.GetInstance(ctx, instanceID)
	if err != nil {
		m.appendLine("failed to get strategy: " + err.Error())
		return
	}

	status := "disabled"
	if instance.Enabled {
		status = "enabled"
	}

	m.appendLine(fmt.Sprintf("strategy: %s", instance.Name))
	m.appendLine(fmt.Sprintf("  id: %s", instance.ID))
	m.appendLine(fmt.Sprintf("  template: %s", instance.TemplateID))
	m.appendLine(fmt.Sprintf("  symbol: %s", instance.Symbol))
	m.appendLine(fmt.Sprintf("  status: %s", status))
	m.appendLine(fmt.Sprintf("  parameters: %d configured", len(instance.Parameters)))
	m.appendLine(fmt.Sprintf("  indicators: %d", len(instance.Indicators)))
	m.appendLine(fmt.Sprintf("  rules: %d", len(instance.Rules)))
}

// handleStrategyUpdate updates a strategy instance.
func (m *Model) handleStrategyUpdate(ctx context.Context, args []string) {
	if len(args) < 2 {
		m.appendLine("usage: /strategy update <instance_id> <param=value...>")
		return
	}

	instanceID := args[0]

	// Get existing instance
	instance, err := m.strategyBuilder.GetInstance(ctx, instanceID)
	if err != nil {
		m.appendLine("failed to get strategy: " + err.Error())
		return
	}

	// Get template
	template, err := m.strategyBuilder.GetTemplate(ctx, instance.TemplateID)
	if err != nil {
		m.appendLine("failed to get template: " + err.Error())
		return
	}

	// Parse parameters
	params := make(map[string]interface{})
	for i := 1; i < len(args); i++ {
		parts := strings.SplitN(args[i], "=", 2)
		if len(parts) != 2 {
			m.appendLine("invalid parameter format: " + args[i])
			return
		}

		key := parts[0]
		value := parts[1]

		// Find parameter definition
		var paramDef *builder.ParameterDefinition
		for _, def := range template.Parameters {
			if def.Key == key {
				paramDef = &def
				break
			}
		}

		if paramDef == nil {
			m.appendLine("unknown parameter: " + key)
			return
		}

		// Parse value
		parsedValue, err := m.parseParameterValue(paramDef.Type, value)
		if err != nil {
			m.appendLine(fmt.Sprintf("invalid value for %s: %s", key, err.Error()))
			return
		}

		params[key] = parsedValue
	}

	// Update instance
	updateInput := builder.UpdateInstanceInput{
		Parameters: params,
	}

	_, err = m.strategyBuilder.UpdateInstance(ctx, instanceID, updateInput)
	if err != nil {
		m.appendLine("failed to update strategy: " + err.Error())
		return
	}

	idDisplay := instanceID
	if len(idDisplay) > 8 {
		idDisplay = idDisplay[:8]
	}
	m.appendLine(fmt.Sprintf("strategy updated: %s", idDisplay))
}

// handleStrategyEnable enables a strategy instance.
func (m *Model) handleStrategyEnable(ctx context.Context, args []string) {
	if len(args) == 0 {
		m.appendLine("usage: /strategy enable <instance_id>")
		return
	}

	instanceID := args[0]
	if err := m.strategyBuilder.EnableInstance(ctx, instanceID); err != nil {
		m.appendLine("failed to enable strategy: " + err.Error())
		return
	}

	idDisplay := instanceID
	if len(idDisplay) > 8 {
		idDisplay = idDisplay[:8]
	}
	m.appendLine(fmt.Sprintf("strategy enabled: %s", idDisplay))
}

// handleStrategyDisable disables a strategy instance.
func (m *Model) handleStrategyDisable(ctx context.Context, args []string) {
	if len(args) == 0 {
		m.appendLine("usage: /strategy disable <instance_id>")
		return
	}

	instanceID := args[0]
	if err := m.strategyBuilder.DisableInstance(ctx, instanceID); err != nil {
		m.appendLine("failed to disable strategy: " + err.Error())
		return
	}

	idDisplay := instanceID
	if len(idDisplay) > 8 {
		idDisplay = idDisplay[:8]
	}
	m.appendLine(fmt.Sprintf("strategy disabled: %s", idDisplay))
}

// handleStrategyDelete deletes a strategy instance.
func (m *Model) handleStrategyDelete(ctx context.Context, args []string) {
	if len(args) == 0 {
		m.appendLine("usage: /strategy delete <instance_id>")
		return
	}

	instanceID := args[0]
	if err := m.strategyBuilder.DeleteInstance(ctx, instanceID); err != nil {
		m.appendLine("failed to delete strategy: " + err.Error())
		return
	}

	idDisplay := instanceID
	if len(idDisplay) > 8 {
		idDisplay = idDisplay[:8]
	}
	m.appendLine(fmt.Sprintf("strategy deleted: %s", idDisplay))
}

// parseParameterValue parses a string value into the appropriate type.
func (m *Model) parseParameterValue(paramType builder.ParameterType, value string) (interface{}, error) {
	switch paramType {
	case builder.ParameterTypeInt:
		return strconv.Atoi(value)
	case builder.ParameterTypeFloat:
		return strconv.ParseFloat(value, 64)
	case builder.ParameterTypeDecimal:
		return decimal.NewFromString(value)
	case builder.ParameterTypeString:
		return value, nil
	case builder.ParameterTypeBool:
		return strconv.ParseBool(value)
	case builder.ParameterTypeEnum:
		return value, nil
	default:
		return nil, fmt.Errorf("unsupported parameter type: %s", paramType)
	}
}
