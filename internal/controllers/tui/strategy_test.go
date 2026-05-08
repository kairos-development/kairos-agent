package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/kairos-development/kairos-agent/internal/strategy/builder"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockStrategyBuilderService struct {
	templates      []*builder.StrategyTemplate
	instances      []*builder.StrategyInstance
	createErr      error
	updateErr      error
	enableErr      error
	disableErr     error
	deleteErr      error
	getInstanceErr error
	getTemplateErr error
}

func (m *mockStrategyBuilderService) ListTemplates(context.Context) ([]*builder.StrategyTemplate, error) {
	return m.templates, nil
}

func (m *mockStrategyBuilderService) GetTemplate(_ context.Context, id string) (*builder.StrategyTemplate, error) {
	if m.getTemplateErr != nil {
		return nil, m.getTemplateErr
	}
	for _, tmpl := range m.templates {
		if tmpl.ID == id {
			return tmpl, nil
		}
	}
	return nil, assert.AnError
}

func (m *mockStrategyBuilderService) CreateInstance(_ context.Context, input builder.CreateInstanceInput) (*builder.StrategyInstance, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	instance := &builder.StrategyInstance{
		ID:         "test-instance-id",
		TemplateID: input.TemplateID,
		Name:       input.Name,
		Symbol:     input.Symbol,
		Parameters: input.Parameters,
	}
	m.instances = append(m.instances, instance)
	return instance, nil
}

func (m *mockStrategyBuilderService) UpdateInstance(_ context.Context, id string, input builder.UpdateInstanceInput) (*builder.StrategyInstance, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	for _, inst := range m.instances {
		if inst.ID == id {
			if input.Parameters != nil {
				inst.Parameters = input.Parameters
			}
			return inst, nil
		}
	}
	return nil, assert.AnError
}

func (m *mockStrategyBuilderService) GetInstance(_ context.Context, id string) (*builder.StrategyInstance, error) {
	if m.getInstanceErr != nil {
		return nil, m.getInstanceErr
	}
	for _, inst := range m.instances {
		if inst.ID == id {
			return inst, nil
		}
	}
	return nil, assert.AnError
}

func (m *mockStrategyBuilderService) ListInstances(context.Context) ([]*builder.StrategyInstance, error) {
	return m.instances, nil
}

func (m *mockStrategyBuilderService) DeleteInstance(_ context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	for i, inst := range m.instances {
		if inst.ID == id {
			m.instances = append(m.instances[:i], m.instances[i+1:]...)
			return nil
		}
	}
	return assert.AnError
}

func (m *mockStrategyBuilderService) EnableInstance(_ context.Context, id string) error {
	if m.enableErr != nil {
		return m.enableErr
	}
	for _, inst := range m.instances {
		if inst.ID == id {
			inst.Enabled = true
			return nil
		}
	}
	return assert.AnError
}

func (m *mockStrategyBuilderService) DisableInstance(_ context.Context, id string) error {
	if m.disableErr != nil {
		return m.disableErr
	}
	for _, inst := range m.instances {
		if inst.ID == id {
			inst.Enabled = false
			return nil
		}
	}
	return assert.AnError
}

func TestModel_HandleStrategyCommand_Templates(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{
		templates: []*builder.StrategyTemplate{
			{
				ID:       "sma_crossover",
				Name:     "SMA Crossover",
				Category: builder.CategoryTrend,
			},
			{
				ID:       "rsi_mean_revert",
				Name:     "RSI Mean Reversion",
				Category: builder.CategoryMeanRevert,
			},
		},
	}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy templates")

	assert.Contains(t, model.lines[len(model.lines)-3], "available templates")
	assert.Contains(t, model.lines[len(model.lines)-2], "sma_crossover")
	assert.Contains(t, model.lines[len(model.lines)-1], "rsi_mean_revert")
}

func TestModel_HandleStrategyCommand_List_Empty(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{
		instances: []*builder.StrategyInstance{},
	}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy list")

	assert.Contains(t, model.lines[len(model.lines)-1], "no strategies configured")
}

func TestModel_HandleStrategyCommand_List(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{
		instances: []*builder.StrategyInstance{
			{
				ID:         "instance-1",
				Name:       "My Strategy",
				TemplateID: "sma_crossover",
				Symbol:     "BTCUSDT",
				Enabled:    true,
			},
		},
	}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy list")

	assert.Contains(t, model.lines[len(model.lines)-2], "configured strategies")
	assert.Contains(t, model.lines[len(model.lines)-1], "My Strategy")
	assert.Contains(t, model.lines[len(model.lines)-1], "enabled")
}

func TestModel_HandleStrategyCommand_Create(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{
		templates: []*builder.StrategyTemplate{
			{
				ID:   "sma_crossover",
				Name: "SMA Crossover",
				Parameters: []builder.ParameterDefinition{
					{Key: "fast_period", Type: builder.ParameterTypeInt},
					{Key: "slow_period", Type: builder.ParameterTypeInt},
					{Key: "quantity", Type: builder.ParameterTypeDecimal},
				},
			},
		},
	}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy create sma_crossover MyStrategy BTCUSDT fast_period=10 slow_period=30 quantity=0.01")

	assert.Contains(t, model.lines[len(model.lines)-1], "strategy created")
	assert.Len(t, strategySvc.instances, 1)
	assert.Equal(t, "MyStrategy", strategySvc.instances[0].Name)
}

func TestModel_HandleStrategyCommand_Create_MissingArgs(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy create sma_crossover")

	assert.Contains(t, model.lines[len(model.lines)-1], "usage:")
}

func TestModel_HandleStrategyCommand_Create_InvalidTemplate(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{
		getTemplateErr: assert.AnError,
	}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy create invalid_template MyStrategy BTCUSDT")

	assert.Contains(t, model.lines[len(model.lines)-1], "failed to get template")
}

func TestModel_HandleStrategyCommand_Show(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{
		instances: []*builder.StrategyInstance{
			{
				ID:         "test-id",
				Name:       "Test Strategy",
				TemplateID: "sma_crossover",
				Symbol:     "BTCUSDT",
				Enabled:    true,
				Parameters: map[string]interface{}{"fast_period": 10},
				Indicators: []builder.IndicatorConfig{{}},
				Rules:      []builder.RuleConfig{{}, {}},
			},
		},
	}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy show test-id")

	// Find the lines containing the expected output
	output := strings.Join(model.lines, "\n")
	assert.Contains(t, output, "strategy: Test Strategy")
	assert.Contains(t, output, "id: test-id")
	assert.Contains(t, output, "template: sma_crossover")
	assert.Contains(t, output, "symbol: BTCUSDT")
	assert.Contains(t, output, "status: enabled")
	assert.Contains(t, output, "parameters: 1 configured")
	assert.Contains(t, output, "rules: 2")
}

func TestModel_HandleStrategyCommand_Update(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{
		templates: []*builder.StrategyTemplate{
			{
				ID: "sma_crossover",
				Parameters: []builder.ParameterDefinition{
					{Key: "fast_period", Type: builder.ParameterTypeInt},
				},
			},
		},
		instances: []*builder.StrategyInstance{
			{
				ID:         "test-id",
				TemplateID: "sma_crossover",
				Parameters: map[string]interface{}{"fast_period": 10},
			},
		},
	}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy update test-id fast_period=20")

	assert.Contains(t, model.lines[len(model.lines)-1], "strategy updated")
	assert.Equal(t, 20, strategySvc.instances[0].Parameters["fast_period"])
}

func TestModel_HandleStrategyCommand_Enable(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{
		instances: []*builder.StrategyInstance{
			{ID: "test-id", Enabled: false},
		},
	}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy enable test-id")

	assert.Contains(t, model.lines[len(model.lines)-1], "strategy enabled")
	assert.True(t, strategySvc.instances[0].Enabled)
}

func TestModel_HandleStrategyCommand_Disable(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{
		instances: []*builder.StrategyInstance{
			{ID: "test-id", Enabled: true},
		},
	}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy disable test-id")

	assert.Contains(t, model.lines[len(model.lines)-1], "strategy disabled")
	assert.False(t, strategySvc.instances[0].Enabled)
}

func TestModel_HandleStrategyCommand_Delete(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{
		instances: []*builder.StrategyInstance{
			{ID: "test-id"},
		},
	}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy delete test-id")

	assert.Contains(t, model.lines[len(model.lines)-1], "strategy deleted")
	assert.Empty(t, strategySvc.instances)
}

func TestModel_HandleStrategyCommand_NoArgs(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy")

	assert.Contains(t, model.lines[len(model.lines)-1], "usage:")
}

func TestModel_HandleStrategyCommand_UnknownSubcommand(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy unknown")

	assert.Contains(t, model.lines[len(model.lines)-1], "unknown strategy subcommand")
}

func TestModel_HandleStrategyCommand_NotAvailable(t *testing.T) {
	svc := &mockAgentService{}
	model := New(svc)

	model.handleCommand("/strategy list")

	assert.Contains(t, model.lines[len(model.lines)-1], "strategy builder not available")
}

func TestModel_ParseParameterValue(t *testing.T) {
	model := Model{}

	tests := []struct {
		name      string
		paramType builder.ParameterType
		value     string
		expected  interface{}
		expectErr bool
	}{
		{
			name:      "int",
			paramType: builder.ParameterTypeInt,
			value:     "42",
			expected:  42,
			expectErr: false,
		},
		{
			name:      "float",
			paramType: builder.ParameterTypeFloat,
			value:     "3.14",
			expected:  3.14,
			expectErr: false,
		},
		{
			name:      "decimal",
			paramType: builder.ParameterTypeDecimal,
			value:     "0.01",
			expected:  decimal.NewFromFloat(0.01),
			expectErr: false,
		},
		{
			name:      "string",
			paramType: builder.ParameterTypeString,
			value:     "test",
			expected:  "test",
			expectErr: false,
		},
		{
			name:      "bool true",
			paramType: builder.ParameterTypeBool,
			value:     "true",
			expected:  true,
			expectErr: false,
		},
		{
			name:      "bool false",
			paramType: builder.ParameterTypeBool,
			value:     "false",
			expected:  false,
			expectErr: false,
		},
		{
			name:      "enum",
			paramType: builder.ParameterTypeEnum,
			value:     "option1",
			expected:  "option1",
			expectErr: false,
		},
		{
			name:      "invalid int",
			paramType: builder.ParameterTypeInt,
			value:     "not_an_int",
			expectErr: true,
		},
		{
			name:      "invalid float",
			paramType: builder.ParameterTypeFloat,
			value:     "not_a_float",
			expectErr: true,
		},
		{
			name:      "invalid bool",
			paramType: builder.ParameterTypeBool,
			value:     "not_a_bool",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := model.parseParameterValue(tt.paramType, tt.value)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.paramType == builder.ParameterTypeDecimal {
					assert.True(t, tt.expected.(decimal.Decimal).Equal(result.(decimal.Decimal)))
				} else {
					assert.Equal(t, tt.expected, result)
				}
			}
		})
	}
}

func TestModel_HandleStrategyCommand_Templates_Empty(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{
		templates: []*builder.StrategyTemplate{},
	}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy templates")

	assert.Contains(t, model.lines[len(model.lines)-1], "no templates available")
}

func TestModel_HandleStrategyCommand_Enable_MissingArgs(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy enable")

	assert.Contains(t, model.lines[len(model.lines)-1], "usage:")
}

func TestModel_HandleStrategyCommand_Enable_Error(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{
		enableErr: assert.AnError,
	}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy enable test-id")

	assert.Contains(t, model.lines[len(model.lines)-1], "failed to enable strategy")
}

func TestModel_HandleStrategyCommand_Disable_MissingArgs(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy disable")

	assert.Contains(t, model.lines[len(model.lines)-1], "usage:")
}

func TestModel_HandleStrategyCommand_Disable_Error(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{
		disableErr: assert.AnError,
	}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy disable test-id")

	assert.Contains(t, model.lines[len(model.lines)-1], "failed to disable strategy")
}

func TestModel_HandleStrategyCommand_Delete_MissingArgs(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy delete")

	assert.Contains(t, model.lines[len(model.lines)-1], "usage:")
}

func TestModel_HandleStrategyCommand_Delete_Error(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{
		deleteErr: assert.AnError,
	}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy delete test-id")

	assert.Contains(t, model.lines[len(model.lines)-1], "failed to delete strategy")
}

func TestModel_HandleStrategyCommand_Update_MissingArgs(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy update test-id")

	assert.Contains(t, model.lines[len(model.lines)-1], "usage:")
}

func TestModel_HandleStrategyCommand_Update_GetInstanceError(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{
		getInstanceErr: assert.AnError,
	}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy update test-id param=value")

	assert.Contains(t, model.lines[len(model.lines)-1], "failed to get strategy")
}

func TestModel_HandleStrategyCommand_Update_GetTemplateError(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{
		instances: []*builder.StrategyInstance{
			{ID: "test-id", TemplateID: "template-id"},
		},
		getTemplateErr: assert.AnError,
	}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy update test-id param=value")

	assert.Contains(t, model.lines[len(model.lines)-1], "failed to get template")
}

func TestModel_HandleStrategyCommand_Update_InvalidParameterFormat(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{
		templates: []*builder.StrategyTemplate{
			{ID: "template-id"},
		},
		instances: []*builder.StrategyInstance{
			{ID: "test-id", TemplateID: "template-id"},
		},
	}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy update test-id invalid_format")

	assert.Contains(t, model.lines[len(model.lines)-1], "invalid parameter format")
}

func TestModel_HandleStrategyCommand_Update_UnknownParameter(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{
		templates: []*builder.StrategyTemplate{
			{
				ID: "template-id",
				Parameters: []builder.ParameterDefinition{
					{Key: "known_param", Type: builder.ParameterTypeInt},
				},
			},
		},
		instances: []*builder.StrategyInstance{
			{ID: "test-id", TemplateID: "template-id"},
		},
	}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy update test-id unknown_param=value")

	assert.Contains(t, model.lines[len(model.lines)-1], "unknown parameter")
}

func TestModel_HandleStrategyCommand_Update_InvalidValue(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{
		templates: []*builder.StrategyTemplate{
			{
				ID: "template-id",
				Parameters: []builder.ParameterDefinition{
					{Key: "int_param", Type: builder.ParameterTypeInt},
				},
			},
		},
		instances: []*builder.StrategyInstance{
			{ID: "test-id", TemplateID: "template-id"},
		},
	}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy update test-id int_param=not_an_int")

	assert.Contains(t, model.lines[len(model.lines)-1], "invalid value")
}

func TestModel_HandleStrategyCommand_Update_UpdateError(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{
		templates: []*builder.StrategyTemplate{
			{
				ID: "template-id",
				Parameters: []builder.ParameterDefinition{
					{Key: "int_param", Type: builder.ParameterTypeInt},
				},
			},
		},
		instances: []*builder.StrategyInstance{
			{ID: "test-id", TemplateID: "template-id"},
		},
		updateErr: assert.AnError,
	}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy update test-id int_param=42")

	assert.Contains(t, model.lines[len(model.lines)-1], "failed to update strategy")
}

func TestModel_HandleStrategyCommand_Show_MissingArgs(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy show")

	assert.Contains(t, model.lines[len(model.lines)-1], "usage:")
}

func TestModel_HandleStrategyCommand_Show_Error(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{
		getInstanceErr: assert.AnError,
	}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy show test-id")

	assert.Contains(t, model.lines[len(model.lines)-1], "failed to get strategy")
}

func TestModel_HandleStrategyCommand_Create_InvalidParameterFormat(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{
		templates: []*builder.StrategyTemplate{
			{ID: "template-id"},
		},
	}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy create template-id MyStrategy BTCUSDT invalid_format")

	assert.Contains(t, model.lines[len(model.lines)-1], "invalid parameter format")
}

func TestModel_HandleStrategyCommand_Create_UnknownParameter(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{
		templates: []*builder.StrategyTemplate{
			{
				ID: "template-id",
				Parameters: []builder.ParameterDefinition{
					{Key: "known_param", Type: builder.ParameterTypeInt},
				},
			},
		},
	}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy create template-id MyStrategy BTCUSDT unknown_param=value")

	assert.Contains(t, model.lines[len(model.lines)-1], "unknown parameter")
}

func TestModel_HandleStrategyCommand_Create_InvalidValue(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{
		templates: []*builder.StrategyTemplate{
			{
				ID: "template-id",
				Parameters: []builder.ParameterDefinition{
					{Key: "int_param", Type: builder.ParameterTypeInt},
				},
			},
		},
	}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy create template-id MyStrategy BTCUSDT int_param=not_an_int")

	assert.Contains(t, model.lines[len(model.lines)-1], "invalid value")
}

func TestModel_HandleStrategyCommand_Create_CreateError(t *testing.T) {
	svc := &mockAgentService{}
	strategySvc := &mockStrategyBuilderService{
		templates: []*builder.StrategyTemplate{
			{ID: "template-id"},
		},
		createErr: assert.AnError,
	}
	model := NewWithStrategyBuilder(svc, strategySvc)

	model.handleCommand("/strategy create template-id MyStrategy BTCUSDT")

	assert.Contains(t, model.lines[len(model.lines)-1], "failed to create strategy")
}
