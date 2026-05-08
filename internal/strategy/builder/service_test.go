package builder

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	svc := NewService()
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.library)
	assert.NotNil(t, svc.instances)
}

func TestService_ListTemplates(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	templates, err := svc.ListTemplates(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, templates)
	assert.GreaterOrEqual(t, len(templates), 2) // At least SMA crossover and RSI mean revert
}

func TestService_GetTemplate(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	template, err := svc.GetTemplate(ctx, "sma_crossover")
	require.NoError(t, err)
	assert.Equal(t, "sma_crossover", template.ID)
	assert.Equal(t, "SMA Crossover", template.Name)
	assert.Equal(t, CategoryTrend, template.Category)
	assert.NotEmpty(t, template.Parameters)
	assert.NotEmpty(t, template.Indicators)
	assert.NotEmpty(t, template.Rules)
}

func TestService_GetTemplate_NotFound(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	_, err := svc.GetTemplate(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_CreateInstance(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	input := CreateInstanceInput{
		TemplateID:  "sma_crossover",
		Name:        "My SMA Strategy",
		Description: "Test strategy",
		Symbol:      "BTCUSDT",
		Parameters: map[string]interface{}{
			"fast_period": 10,
			"slow_period": 30,
			"quantity":    decimal.NewFromFloat(0.01),
		},
	}

	instance, err := svc.CreateInstance(ctx, input)
	require.NoError(t, err)
	assert.NotEmpty(t, instance.ID)
	assert.Equal(t, "sma_crossover", instance.TemplateID)
	assert.Equal(t, "My SMA Strategy", instance.Name)
	assert.Equal(t, "BTCUSDT", instance.Symbol)
	assert.False(t, instance.Enabled)
	assert.NotEmpty(t, instance.Parameters)
	assert.NotEmpty(t, instance.Indicators)
	assert.NotEmpty(t, instance.Rules)
	assert.NotZero(t, instance.CreatedAt)
	assert.NotZero(t, instance.UpdatedAt)
}

func TestService_CreateInstance_InvalidTemplate(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	input := CreateInstanceInput{
		TemplateID: "nonexistent",
		Name:       "Test",
		Symbol:     "BTCUSDT",
		Parameters: map[string]interface{}{},
	}

	_, err := svc.CreateInstance(ctx, input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_CreateInstance_MissingRequiredParameter(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	input := CreateInstanceInput{
		TemplateID: "sma_crossover",
		Name:       "Test",
		Symbol:     "BTCUSDT",
		Parameters: map[string]interface{}{
			"fast_period": 10,
			// Missing slow_period and quantity
		},
	}

	_, err := svc.CreateInstance(ctx, input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required parameter")
}

func TestService_CreateInstance_InvalidParameterType(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	input := CreateInstanceInput{
		TemplateID: "sma_crossover",
		Name:       "Test",
		Symbol:     "BTCUSDT",
		Parameters: map[string]interface{}{
			"fast_period": "not_an_int",
			"slow_period": 30,
			"quantity":    decimal.NewFromFloat(0.01),
		},
	}

	_, err := svc.CreateInstance(ctx, input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected int")
}

func TestService_CreateInstance_ParameterOutOfRange(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	input := CreateInstanceInput{
		TemplateID: "sma_crossover",
		Name:       "Test",
		Symbol:     "BTCUSDT",
		Parameters: map[string]interface{}{
			"fast_period": 3, // Below minimum of 5
			"slow_period": 30,
			"quantity":    decimal.NewFromFloat(0.01),
		},
	}

	_, err := svc.CreateInstance(ctx, input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "less than minimum")
}

func TestService_GetInstance(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	input := CreateInstanceInput{
		TemplateID: "sma_crossover",
		Name:       "Test",
		Symbol:     "BTCUSDT",
		Parameters: map[string]interface{}{
			"fast_period": 10,
			"slow_period": 30,
			"quantity":    decimal.NewFromFloat(0.01),
		},
	}

	created, err := svc.CreateInstance(ctx, input)
	require.NoError(t, err)

	retrieved, err := svc.GetInstance(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, retrieved.ID)
	assert.Equal(t, created.Name, retrieved.Name)
}

func TestService_GetInstance_NotFound(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	_, err := svc.GetInstance(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_ListInstances(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	// Initially empty
	instances, err := svc.ListInstances(ctx)
	require.NoError(t, err)
	assert.Empty(t, instances)

	// Create instances
	input1 := CreateInstanceInput{
		TemplateID: "sma_crossover",
		Name:       "Strategy 1",
		Symbol:     "BTCUSDT",
		Parameters: map[string]interface{}{
			"fast_period": 10,
			"slow_period": 30,
			"quantity":    decimal.NewFromFloat(0.01),
		},
	}
	_, err = svc.CreateInstance(ctx, input1)
	require.NoError(t, err)

	input2 := CreateInstanceInput{
		TemplateID: "rsi_mean_revert",
		Name:       "Strategy 2",
		Symbol:     "ETHUSDT",
		Parameters: map[string]interface{}{
			"rsi_period":       14,
			"oversold_level":   30,
			"overbought_level": 70,
			"quantity":         decimal.NewFromFloat(0.1),
		},
	}
	_, err = svc.CreateInstance(ctx, input2)
	require.NoError(t, err)

	instances, err = svc.ListInstances(ctx)
	require.NoError(t, err)
	assert.Len(t, instances, 2)
}

func TestService_UpdateInstance(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	input := CreateInstanceInput{
		TemplateID: "sma_crossover",
		Name:       "Original Name",
		Symbol:     "BTCUSDT",
		Parameters: map[string]interface{}{
			"fast_period": 10,
			"slow_period": 30,
			"quantity":    decimal.NewFromFloat(0.01),
		},
	}

	created, err := svc.CreateInstance(ctx, input)
	require.NoError(t, err)

	newName := "Updated Name"
	newSymbol := "ETHUSDT"
	enabled := true
	updateInput := UpdateInstanceInput{
		Name:    &newName,
		Symbol:  &newSymbol,
		Enabled: &enabled,
	}

	updated, err := svc.UpdateInstance(ctx, created.ID, updateInput)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.Equal(t, "ETHUSDT", updated.Symbol)
	assert.True(t, updated.Enabled)
	assert.True(t, updated.UpdatedAt.After(created.UpdatedAt) || updated.UpdatedAt.Equal(created.UpdatedAt))
}

func TestService_UpdateInstance_Parameters(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	input := CreateInstanceInput{
		TemplateID: "sma_crossover",
		Name:       "Test",
		Symbol:     "BTCUSDT",
		Parameters: map[string]interface{}{
			"fast_period": 10,
			"slow_period": 30,
			"quantity":    decimal.NewFromFloat(0.01),
		},
	}

	created, err := svc.CreateInstance(ctx, input)
	require.NoError(t, err)

	updateInput := UpdateInstanceInput{
		Parameters: map[string]interface{}{
			"fast_period": 15,
			"slow_period": 50,
			"quantity":    decimal.NewFromFloat(0.02),
		},
	}

	updated, err := svc.UpdateInstance(ctx, created.ID, updateInput)
	require.NoError(t, err)
	assert.Equal(t, 15, updated.Parameters["fast_period"])
	assert.Equal(t, 50, updated.Parameters["slow_period"])
}

func TestService_UpdateInstance_NotFound(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	updateInput := UpdateInstanceInput{}
	_, err := svc.UpdateInstance(ctx, "nonexistent", updateInput)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_DeleteInstance(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	input := CreateInstanceInput{
		TemplateID: "sma_crossover",
		Name:       "Test",
		Symbol:     "BTCUSDT",
		Parameters: map[string]interface{}{
			"fast_period": 10,
			"slow_period": 30,
			"quantity":    decimal.NewFromFloat(0.01),
		},
	}

	created, err := svc.CreateInstance(ctx, input)
	require.NoError(t, err)

	err = svc.DeleteInstance(ctx, created.ID)
	require.NoError(t, err)

	_, err = svc.GetInstance(ctx, created.ID)
	assert.Error(t, err)
}

func TestService_DeleteInstance_NotFound(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	err := svc.DeleteInstance(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_EnableInstance(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	input := CreateInstanceInput{
		TemplateID: "sma_crossover",
		Name:       "Test",
		Symbol:     "BTCUSDT",
		Parameters: map[string]interface{}{
			"fast_period": 10,
			"slow_period": 30,
			"quantity":    decimal.NewFromFloat(0.01),
		},
	}

	created, err := svc.CreateInstance(ctx, input)
	require.NoError(t, err)
	assert.False(t, created.Enabled)

	err = svc.EnableInstance(ctx, created.ID)
	require.NoError(t, err)

	retrieved, err := svc.GetInstance(ctx, created.ID)
	require.NoError(t, err)
	assert.True(t, retrieved.Enabled)
}

func TestService_DisableInstance(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	input := CreateInstanceInput{
		TemplateID: "sma_crossover",
		Name:       "Test",
		Symbol:     "BTCUSDT",
		Parameters: map[string]interface{}{
			"fast_period": 10,
			"slow_period": 30,
			"quantity":    decimal.NewFromFloat(0.01),
		},
	}

	created, err := svc.CreateInstance(ctx, input)
	require.NoError(t, err)

	err = svc.EnableInstance(ctx, created.ID)
	require.NoError(t, err)

	err = svc.DisableInstance(ctx, created.ID)
	require.NoError(t, err)

	retrieved, err := svc.GetInstance(ctx, created.ID)
	require.NoError(t, err)
	assert.False(t, retrieved.Enabled)
}

func TestService_TemplateVariableResolution(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	input := CreateInstanceInput{
		TemplateID: "sma_crossover",
		Name:       "Test",
		Symbol:     "BTCUSDT",
		Parameters: map[string]interface{}{
			"fast_period": 15,
			"slow_period": 45,
			"quantity":    decimal.NewFromFloat(0.05),
		},
	}

	instance, err := svc.CreateInstance(ctx, input)
	require.NoError(t, err)

	// Check that template variables were resolved
	assert.Len(t, instance.Indicators, 2)
	assert.Equal(t, 15, instance.Indicators[0].Parameters["period"])
	assert.Equal(t, 45, instance.Indicators[1].Parameters["period"])

	// Check rules
	assert.Len(t, instance.Rules, 2)
	assert.Equal(t, decimal.NewFromFloat(0.05), instance.Rules[0].Action.Quantity.Value)
}

func TestStrategyInstance_Validate(t *testing.T) {
	tests := []struct {
		name        string
		instance    *StrategyInstance
		expectError bool
		errorText   string
	}{
		{
			name: "valid instance",
			instance: &StrategyInstance{
				ID:     "test-id",
				Name:   "Test Strategy",
				Symbol: "BTCUSDT",
				Rules:  []RuleConfig{{}},
			},
			expectError: false,
		},
		{
			name: "missing ID",
			instance: &StrategyInstance{
				Name:   "Test Strategy",
				Symbol: "BTCUSDT",
				Rules:  []RuleConfig{{}},
			},
			expectError: true,
			errorText:   "ID is required",
		},
		{
			name: "missing name",
			instance: &StrategyInstance{
				ID:     "test-id",
				Symbol: "BTCUSDT",
				Rules:  []RuleConfig{{}},
			},
			expectError: true,
			errorText:   "name is required",
		},
		{
			name: "missing symbol",
			instance: &StrategyInstance{
				ID:    "test-id",
				Name:  "Test Strategy",
				Rules: []RuleConfig{{}},
			},
			expectError: true,
			errorText:   "symbol is required",
		},
		{
			name: "no rules",
			instance: &StrategyInstance{
				ID:     "test-id",
				Name:   "Test Strategy",
				Symbol: "BTCUSDT",
				Rules:  []RuleConfig{},
			},
			expectError: true,
			errorText:   "at least one rule is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.instance.Validate()
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorText)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTemplateLibrary_RSIMeanRevert(t *testing.T) {
	lib := NewTemplateLibrary()

	template, err := lib.GetTemplate("rsi_mean_revert")
	require.NoError(t, err)
	assert.Equal(t, "rsi_mean_revert", template.ID)
	assert.Equal(t, "RSI Mean Reversion", template.Name)
	assert.Equal(t, CategoryMeanRevert, template.Category)
	assert.Len(t, template.Parameters, 4)
	assert.Len(t, template.Indicators, 1)
	assert.Len(t, template.Rules, 2)
}

func TestTemplateLibrary_AddTemplate(t *testing.T) {
	lib := NewTemplateLibrary()

	customTemplate := &StrategyTemplate{
		ID:          "custom_strategy",
		Name:        "Custom Strategy",
		Description: "A custom strategy",
		Category:    CategoryCustom,
		Parameters:  []ParameterDefinition{},
		Indicators:  []IndicatorConfig{},
		Rules:       []RuleConfig{{}},
	}

	err := lib.AddTemplate(customTemplate)
	require.NoError(t, err)

	retrieved, err := lib.GetTemplate("custom_strategy")
	require.NoError(t, err)
	assert.Equal(t, "custom_strategy", retrieved.ID)
}

func TestTemplateLibrary_AddTemplate_NoID(t *testing.T) {
	lib := NewTemplateLibrary()

	customTemplate := &StrategyTemplate{
		Name: "Custom Strategy",
	}

	err := lib.AddTemplate(customTemplate)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ID is required")
}

func TestTemplateLibrary_AddTemplate_Duplicate(t *testing.T) {
	lib := NewTemplateLibrary()

	customTemplate := &StrategyTemplate{
		ID:   "sma_crossover",
		Name: "Duplicate",
	}

	err := lib.AddTemplate(customTemplate)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestService_ValidateParameterValue_Float(t *testing.T) {
	svc := NewService()

	def := ParameterDefinition{
		Type:     ParameterTypeFloat,
		MinValue: 0.1,
		MaxValue: 10.0,
	}

	// Valid value
	err := svc.validateParameterValue(def, 5.0)
	assert.NoError(t, err)

	// Below minimum
	err = svc.validateParameterValue(def, 0.05)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "less than minimum")

	// Above maximum
	err = svc.validateParameterValue(def, 15.0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "greater than maximum")

	// Wrong type
	err = svc.validateParameterValue(def, "not_a_float")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected float64")
}

func TestService_ValidateParameterValue_String(t *testing.T) {
	svc := NewService()

	def := ParameterDefinition{
		Type: ParameterTypeString,
	}

	err := svc.validateParameterValue(def, "test_string")
	assert.NoError(t, err)

	err = svc.validateParameterValue(def, 123)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected string")
}

func TestService_ValidateParameterValue_Bool(t *testing.T) {
	svc := NewService()

	def := ParameterDefinition{
		Type: ParameterTypeBool,
	}

	err := svc.validateParameterValue(def, true)
	assert.NoError(t, err)

	err = svc.validateParameterValue(def, "not_a_bool")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected bool")
}

func TestService_ValidateParameterValue_Enum(t *testing.T) {
	svc := NewService()

	def := ParameterDefinition{
		Type:    ParameterTypeEnum,
		Options: []string{"option1", "option2", "option3"},
	}

	err := svc.validateParameterValue(def, "option1")
	assert.NoError(t, err)

	err = svc.validateParameterValue(def, "invalid_option")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a valid option")

	err = svc.validateParameterValue(def, 123)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected string")
}

func TestService_ValidateParameterValue_Decimal(t *testing.T) {
	svc := NewService()

	def := ParameterDefinition{
		Type:     ParameterTypeDecimal,
		MinValue: decimal.NewFromFloat(0.001),
		MaxValue: decimal.NewFromFloat(100.0),
	}

	// Valid value
	err := svc.validateParameterValue(def, decimal.NewFromFloat(50.0))
	assert.NoError(t, err)

	// Below minimum
	err = svc.validateParameterValue(def, decimal.NewFromFloat(0.0001))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "less than minimum")

	// Above maximum
	err = svc.validateParameterValue(def, decimal.NewFromFloat(200.0))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "greater than maximum")

	// Wrong type
	err = svc.validateParameterValue(def, "not_a_decimal")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected decimal.Decimal")
}

func TestService_ResolveCondition_Nested(t *testing.T) {
	svc := NewService()

	condition := Condition{
		Operator: OperatorAnd,
		Conditions: []Condition{
			{
				Operator: OperatorGreaterThan,
				Left:     ValueSource{Type: ValueSourceIndicator, Value: "rsi"},
				Right:    ValueSource{Type: ValueSourceStatic, Value: "{{threshold}}"},
			},
			{
				Operator: OperatorLessThan,
				Left:     ValueSource{Type: ValueSourcePrice, Value: "last"},
				Right:    ValueSource{Type: ValueSourceStatic, Value: "{{max_price}}"},
			},
		},
	}

	params := map[string]interface{}{
		"threshold": 70,
		"max_price": 50000,
	}

	resolved := svc.resolveCondition(condition, params)
	assert.Equal(t, OperatorAnd, resolved.Operator)
	assert.Len(t, resolved.Conditions, 2)
	assert.Equal(t, 70, resolved.Conditions[0].Right.Value)
	assert.Equal(t, 50000, resolved.Conditions[1].Right.Value)
}

func TestService_ResolveValue_NonTemplate(t *testing.T) {
	svc := NewService()

	// Non-string value
	result := svc.resolveValue(123, map[string]interface{}{})
	assert.Equal(t, 123, result)

	// String without template syntax
	result = svc.resolveValue("plain_string", map[string]interface{}{})
	assert.Equal(t, "plain_string", result)

	// Template variable not in params
	result = svc.resolveValue("{{missing}}", map[string]interface{}{})
	assert.Equal(t, "{{missing}}", result)
}

func TestService_CreateInstance_EmptyName(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	input := CreateInstanceInput{
		TemplateID: "sma_crossover",
		Name:       "",
		Symbol:     "BTCUSDT",
		Parameters: map[string]interface{}{
			"fast_period": 10,
			"slow_period": 30,
			"quantity":    decimal.NewFromFloat(0.01),
		},
	}

	_, err := svc.CreateInstance(ctx, input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestService_CreateInstance_EmptySymbol(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	input := CreateInstanceInput{
		TemplateID: "sma_crossover",
		Name:       "Test",
		Symbol:     "",
		Parameters: map[string]interface{}{
			"fast_period": 10,
			"slow_period": 30,
			"quantity":    decimal.NewFromFloat(0.01),
		},
	}

	_, err := svc.CreateInstance(ctx, input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "symbol is required")
}

func TestService_UpdateInstance_InvalidParameters(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	input := CreateInstanceInput{
		TemplateID: "sma_crossover",
		Name:       "Test",
		Symbol:     "BTCUSDT",
		Parameters: map[string]interface{}{
			"fast_period": 10,
			"slow_period": 30,
			"quantity":    decimal.NewFromFloat(0.01),
		},
	}

	created, err := svc.CreateInstance(ctx, input)
	require.NoError(t, err)

	updateInput := UpdateInstanceInput{
		Parameters: map[string]interface{}{
			"fast_period": 3, // Below minimum
			"slow_period": 30,
			"quantity":    decimal.NewFromFloat(0.01),
		},
	}

	_, err = svc.UpdateInstance(ctx, created.ID, updateInput)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "less than minimum")
}

func TestService_EnableInstance_NotFound(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	err := svc.EnableInstance(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_DisableInstance_NotFound(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	err := svc.DisableInstance(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
