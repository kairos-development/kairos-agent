package builder

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Service manages strategy building and configuration.
type Service struct {
	library   *TemplateLibrary
	instances map[string]*StrategyInstance
}

// NewService creates a new strategy builder service.
func NewService() *Service {
	return &Service{
		library:   NewTemplateLibrary(),
		instances: make(map[string]*StrategyInstance),
	}
}

// ListTemplates returns all available strategy templates.
func (s *Service) ListTemplates(ctx context.Context) ([]*StrategyTemplate, error) {
	return s.library.ListTemplates(), nil
}

// GetTemplate retrieves a specific template by ID.
func (s *Service) GetTemplate(ctx context.Context, templateID string) (*StrategyTemplate, error) {
	return s.library.GetTemplate(templateID)
}

// CreateInstance creates a new strategy instance from a template.
func (s *Service) CreateInstance(ctx context.Context, input CreateInstanceInput) (*StrategyInstance, error) {
	template, err := s.library.GetTemplate(input.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}

	// Validate parameters
	if err := s.validateParameters(template.Parameters, input.Parameters); err != nil {
		return nil, fmt.Errorf("validate parameters: %w", err)
	}

	// Create instance
	instance := &StrategyInstance{
		ID:          uuid.New().String(),
		TemplateID:  input.TemplateID,
		Name:        input.Name,
		Description: input.Description,
		Symbol:      input.Symbol,
		Enabled:     false,
		Parameters:  input.Parameters,
		Indicators:  s.resolveIndicators(template.Indicators, input.Parameters),
		Rules:       s.resolveRules(template.Rules, input.Parameters),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := instance.Validate(); err != nil {
		return nil, fmt.Errorf("validate instance: %w", err)
	}

	s.instances[instance.ID] = instance
	return instance, nil
}

// UpdateInstance updates an existing strategy instance.
func (s *Service) UpdateInstance(ctx context.Context, instanceID string, input UpdateInstanceInput) (*StrategyInstance, error) {
	instance, exists := s.instances[instanceID]
	if !exists {
		return nil, fmt.Errorf("instance %s not found", instanceID)
	}

	template, err := s.library.GetTemplate(instance.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}

	// Update fields
	if input.Name != nil {
		instance.Name = *input.Name
	}
	if input.Description != nil {
		instance.Description = *input.Description
	}
	if input.Symbol != nil {
		instance.Symbol = *input.Symbol
	}
	if input.Enabled != nil {
		instance.Enabled = *input.Enabled
	}
	if input.Parameters != nil {
		if err := s.validateParameters(template.Parameters, input.Parameters); err != nil {
			return nil, fmt.Errorf("validate parameters: %w", err)
		}
		instance.Parameters = input.Parameters
		instance.Indicators = s.resolveIndicators(template.Indicators, input.Parameters)
		instance.Rules = s.resolveRules(template.Rules, input.Parameters)
	}

	instance.UpdatedAt = time.Now().UTC()

	if err := instance.Validate(); err != nil {
		return nil, fmt.Errorf("validate instance: %w", err)
	}

	return instance, nil
}

// GetInstance retrieves a strategy instance by ID.
func (s *Service) GetInstance(ctx context.Context, instanceID string) (*StrategyInstance, error) {
	instance, exists := s.instances[instanceID]
	if !exists {
		return nil, fmt.Errorf("instance %s not found", instanceID)
	}
	return instance, nil
}

// ListInstances returns all strategy instances.
func (s *Service) ListInstances(ctx context.Context) ([]*StrategyInstance, error) {
	instances := make([]*StrategyInstance, 0, len(s.instances))
	for _, instance := range s.instances {
		instances = append(instances, instance)
	}
	return instances, nil
}

// DeleteInstance deletes a strategy instance.
func (s *Service) DeleteInstance(ctx context.Context, instanceID string) error {
	if _, exists := s.instances[instanceID]; !exists {
		return fmt.Errorf("instance %s not found", instanceID)
	}
	delete(s.instances, instanceID)
	return nil
}

// EnableInstance enables a strategy instance.
func (s *Service) EnableInstance(ctx context.Context, instanceID string) error {
	instance, exists := s.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance %s not found", instanceID)
	}
	instance.Enabled = true
	instance.UpdatedAt = time.Now().UTC()
	return nil
}

// DisableInstance disables a strategy instance.
func (s *Service) DisableInstance(ctx context.Context, instanceID string) error {
	instance, exists := s.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance %s not found", instanceID)
	}
	instance.Enabled = false
	instance.UpdatedAt = time.Now().UTC()
	return nil
}

// validateParameters validates parameter values against definitions.
func (s *Service) validateParameters(definitions []ParameterDefinition, values map[string]interface{}) error {
	for _, def := range definitions {
		value, exists := values[def.Key]
		if !exists {
			if def.Required {
				return fmt.Errorf("required parameter %s is missing", def.Key)
			}
			continue
		}

		if err := s.validateParameterValue(def, value); err != nil {
			return fmt.Errorf("parameter %s: %w", def.Key, err)
		}
	}
	return nil
}

// validateParameterValue validates a single parameter value.
func (s *Service) validateParameterValue(def ParameterDefinition, value interface{}) error {
	switch def.Type {
	case ParameterTypeInt:
		intVal, ok := value.(int)
		if !ok {
			return fmt.Errorf("expected int, got %T", value)
		}
		if def.MinValue != nil {
			if intVal < def.MinValue.(int) {
				return fmt.Errorf("value %d is less than minimum %d", intVal, def.MinValue.(int))
			}
		}
		if def.MaxValue != nil {
			if intVal > def.MaxValue.(int) {
				return fmt.Errorf("value %d is greater than maximum %d", intVal, def.MaxValue.(int))
			}
		}

	case ParameterTypeFloat:
		floatVal, ok := value.(float64)
		if !ok {
			return fmt.Errorf("expected float64, got %T", value)
		}
		if def.MinValue != nil {
			if floatVal < def.MinValue.(float64) {
				return fmt.Errorf("value %f is less than minimum %f", floatVal, def.MinValue.(float64))
			}
		}
		if def.MaxValue != nil {
			if floatVal > def.MaxValue.(float64) {
				return fmt.Errorf("value %f is greater than maximum %f", floatVal, def.MaxValue.(float64))
			}
		}

	case ParameterTypeDecimal:
		decVal, ok := value.(decimal.Decimal)
		if !ok {
			return fmt.Errorf("expected decimal.Decimal, got %T", value)
		}
		if def.MinValue != nil {
			if decVal.LessThan(def.MinValue.(decimal.Decimal)) {
				return fmt.Errorf("value %s is less than minimum %s", decVal.String(), def.MinValue.(decimal.Decimal).String())
			}
		}
		if def.MaxValue != nil {
			if decVal.GreaterThan(def.MaxValue.(decimal.Decimal)) {
				return fmt.Errorf("value %s is greater than maximum %s", decVal.String(), def.MaxValue.(decimal.Decimal).String())
			}
		}

	case ParameterTypeString:
		_, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string, got %T", value)
		}

	case ParameterTypeBool:
		_, ok := value.(bool)
		if !ok {
			return fmt.Errorf("expected bool, got %T", value)
		}

	case ParameterTypeEnum:
		strVal, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
		valid := false
		for _, option := range def.Options {
			if strVal == option {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("value %s is not a valid option", strVal)
		}
	}

	return nil
}

// resolveIndicators resolves template variables in indicator configurations.
func (s *Service) resolveIndicators(indicators []IndicatorConfig, params map[string]interface{}) []IndicatorConfig {
	resolved := make([]IndicatorConfig, len(indicators))
	for i, indicator := range indicators {
		resolved[i] = IndicatorConfig{
			Type:       indicator.Type,
			Parameters: s.resolveMap(indicator.Parameters, params),
			OutputKey:  indicator.OutputKey,
		}
	}
	return resolved
}

// resolveRules resolves template variables in rule configurations.
func (s *Service) resolveRules(rules []RuleConfig, params map[string]interface{}) []RuleConfig {
	resolved := make([]RuleConfig, len(rules))
	for i, rule := range rules {
		resolved[i] = RuleConfig{
			Type:      rule.Type,
			Condition: s.resolveCondition(rule.Condition, params),
			Action:    s.resolveAction(rule.Action, params),
		}
	}
	return resolved
}

// resolveCondition resolves template variables in a condition.
func (s *Service) resolveCondition(condition Condition, params map[string]interface{}) Condition {
	resolved := Condition{
		Operator: condition.Operator,
		Left:     s.resolveValueSource(condition.Left, params),
		Right:    s.resolveValueSource(condition.Right, params),
	}
	if len(condition.Conditions) > 0 {
		resolved.Conditions = make([]Condition, len(condition.Conditions))
		for i, cond := range condition.Conditions {
			resolved.Conditions[i] = s.resolveCondition(cond, params)
		}
	}
	return resolved
}

// resolveAction resolves template variables in an action.
func (s *Service) resolveAction(action ActionConfig, params map[string]interface{}) ActionConfig {
	return ActionConfig{
		Type:       action.Type,
		Side:       action.Side,
		Quantity:   s.resolveQuantityConfig(action.Quantity, params),
		PriceType:  action.PriceType,
		PriceValue: s.resolveValue(action.PriceValue, params),
	}
}

// resolveQuantityConfig resolves template variables in quantity config.
func (s *Service) resolveQuantityConfig(qty QuantityConfig, params map[string]interface{}) QuantityConfig {
	return QuantityConfig{
		Type:  qty.Type,
		Value: s.resolveValue(qty.Value, params),
	}
}

// resolveValueSource resolves template variables in a value source.
func (s *Service) resolveValueSource(source ValueSource, params map[string]interface{}) ValueSource {
	return ValueSource{
		Type:  source.Type,
		Value: s.resolveValue(source.Value, params),
	}
}

// resolveMap resolves template variables in a map.
func (s *Service) resolveMap(m map[string]interface{}, params map[string]interface{}) map[string]interface{} {
	resolved := make(map[string]interface{})
	for k, v := range m {
		resolved[k] = s.resolveValue(v, params)
	}
	return resolved
}

// resolveValue resolves a template variable like {{param_name}}.
func (s *Service) resolveValue(value interface{}, params map[string]interface{}) interface{} {
	strVal, ok := value.(string)
	if !ok {
		return value
	}

	if strings.HasPrefix(strVal, "{{") && strings.HasSuffix(strVal, "}}") {
		paramKey := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strVal, "{{"), "}}"))
		if paramValue, exists := params[paramKey]; exists {
			return paramValue
		}
	}

	return value
}

// CreateInstanceInput is the input for creating a strategy instance.
type CreateInstanceInput struct {
	TemplateID  string
	Name        string
	Description string
	Symbol      string
	Parameters  map[string]interface{}
}

// UpdateInstanceInput is the input for updating a strategy instance.
type UpdateInstanceInput struct {
	Name        *string
	Description *string
	Symbol      *string
	Enabled     *bool
	Parameters  map[string]interface{}
}
