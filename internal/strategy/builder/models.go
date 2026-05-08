package builder

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// StrategyTemplate defines a pre-built strategy template that can be configured.
type StrategyTemplate struct {
	ID          string
	Name        string
	Description string
	Category    StrategyCategory
	Parameters  []ParameterDefinition
	Indicators  []IndicatorConfig
	Rules       []RuleConfig
}

// StrategyCategory categorizes strategy types.
type StrategyCategory string

const (
	CategoryTrend      StrategyCategory = "trend"
	CategoryMeanRevert StrategyCategory = "mean_revert"
	CategoryMomentum   StrategyCategory = "momentum"
	CategoryBreakout   StrategyCategory = "breakout"
	CategoryArbitrage  StrategyCategory = "arbitrage"
	CategoryCustom     StrategyCategory = "custom"
)

// ParameterDefinition defines a configurable parameter.
type ParameterDefinition struct {
	Key          string
	Name         string
	Description  string
	Type         ParameterType
	DefaultValue interface{}
	MinValue     interface{}
	MaxValue     interface{}
	Required     bool
	Options      []string // For enum types
}

// ParameterType defines the type of a parameter.
type ParameterType string

const (
	ParameterTypeInt     ParameterType = "int"
	ParameterTypeFloat   ParameterType = "float"
	ParameterTypeDecimal ParameterType = "decimal"
	ParameterTypeString  ParameterType = "string"
	ParameterTypeBool    ParameterType = "bool"
	ParameterTypeEnum    ParameterType = "enum"
)

// IndicatorConfig defines an indicator used in the strategy.
type IndicatorConfig struct {
	Type       IndicatorType
	Parameters map[string]interface{}
	OutputKey  string // Key to reference this indicator's output
}

// IndicatorType defines available technical indicators.
type IndicatorType string

const (
	IndicatorTypeSMA        IndicatorType = "sma"
	IndicatorTypeEMA        IndicatorType = "ema"
	IndicatorTypeRSI        IndicatorType = "rsi"
	IndicatorTypeMACD       IndicatorType = "macd"
	IndicatorTypeBollinger  IndicatorType = "bollinger"
	IndicatorTypeATR        IndicatorType = "atr"
	IndicatorTypeStochastic IndicatorType = "stochastic"
	IndicatorTypeVolume     IndicatorType = "volume"
)

// RuleConfig defines a trading rule (entry/exit condition).
type RuleConfig struct {
	Type      RuleType
	Condition Condition
	Action    ActionConfig
}

// RuleType defines the type of rule.
type RuleType string

const (
	RuleTypeEntry RuleType = "entry"
	RuleTypeExit  RuleType = "exit"
	RuleTypeStop  RuleType = "stop"
)

// Condition defines a logical condition for a rule.
type Condition struct {
	Operator   ConditionOperator
	Left       ValueSource
	Right      ValueSource
	Conditions []Condition // For AND/OR operators
}

// ConditionOperator defines logical operators.
type ConditionOperator string

const (
	OperatorGreaterThan    ConditionOperator = "gt"
	OperatorLessThan       ConditionOperator = "lt"
	OperatorEquals         ConditionOperator = "eq"
	OperatorGreaterOrEqual ConditionOperator = "gte"
	OperatorLessOrEqual    ConditionOperator = "lte"
	OperatorCrossAbove     ConditionOperator = "cross_above"
	OperatorCrossBelow     ConditionOperator = "cross_below"
	OperatorAnd            ConditionOperator = "and"
	OperatorOr             ConditionOperator = "or"
)

// ValueSource defines where a value comes from.
type ValueSource struct {
	Type  ValueSourceType
	Value interface{} // Static value or reference key
}

// ValueSourceType defines the source of a value.
type ValueSourceType string

const (
	ValueSourceStatic    ValueSourceType = "static"
	ValueSourceIndicator ValueSourceType = "indicator"
	ValueSourcePrice     ValueSourceType = "price"
	ValueSourceVolume    ValueSourceType = "volume"
	ValueSourcePosition  ValueSourceType = "position"
	ValueSourceBalance   ValueSourceType = "balance"
)

// ActionConfig defines the action to take when a rule is triggered.
type ActionConfig struct {
	Type       ActionType
	Side       string // "BUY" or "SELL"
	Quantity   QuantityConfig
	PriceType  PriceType
	PriceValue interface{} // For limit orders
}

// ActionType defines the type of action.
type ActionType string

const (
	ActionTypeMarketOrder   ActionType = "market_order"
	ActionTypeLimitOrder    ActionType = "limit_order"
	ActionTypeClosePosition ActionType = "close_position"
)

// PriceType defines how order price is determined.
type PriceType string

const (
	PriceTypeMarket PriceType = "market"
	PriceTypeLimit  PriceType = "limit"
	PriceTypeStop   PriceType = "stop"
)

// QuantityConfig defines how order quantity is calculated.
type QuantityConfig struct {
	Type  QuantityType
	Value interface{} // Fixed amount or percentage
}

// QuantityType defines how quantity is determined.
type QuantityType string

const (
	QuantityTypeFixed      QuantityType = "fixed"
	QuantityTypePercentage QuantityType = "percentage"
	QuantityTypeRiskBased  QuantityType = "risk_based"
)

// StrategyInstance represents a configured strategy instance.
type StrategyInstance struct {
	ID          string
	TemplateID  string
	Name        string
	Description string
	Symbol      string
	Enabled     bool
	Parameters  map[string]interface{}
	Indicators  []IndicatorConfig
	Rules       []RuleConfig
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Validate validates the strategy instance configuration.
func (si *StrategyInstance) Validate() error {
	if si.ID == "" {
		return fmt.Errorf("strategy ID is required")
	}
	if si.Name == "" {
		return fmt.Errorf("strategy name is required")
	}
	if si.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}
	if len(si.Rules) == 0 {
		return fmt.Errorf("at least one rule is required")
	}
	return nil
}

// TemplateLibrary manages available strategy templates.
type TemplateLibrary struct {
	templates map[string]*StrategyTemplate
}

// NewTemplateLibrary creates a new template library with built-in templates.
func NewTemplateLibrary() *TemplateLibrary {
	lib := &TemplateLibrary{
		templates: make(map[string]*StrategyTemplate),
	}
	lib.loadBuiltInTemplates()
	return lib
}

// loadBuiltInTemplates loads pre-defined strategy templates.
func (tl *TemplateLibrary) loadBuiltInTemplates() {
	// SMA Crossover Strategy
	tl.templates["sma_crossover"] = &StrategyTemplate{
		ID:          "sma_crossover",
		Name:        "SMA Crossover",
		Description: "Buy when fast SMA crosses above slow SMA, sell when it crosses below",
		Category:    CategoryTrend,
		Parameters: []ParameterDefinition{
			{
				Key:          "fast_period",
				Name:         "Fast SMA Period",
				Description:  "Period for fast moving average",
				Type:         ParameterTypeInt,
				DefaultValue: 10,
				MinValue:     5,
				MaxValue:     50,
				Required:     true,
			},
			{
				Key:          "slow_period",
				Name:         "Slow SMA Period",
				Description:  "Period for slow moving average",
				Type:         ParameterTypeInt,
				DefaultValue: 30,
				MinValue:     20,
				MaxValue:     200,
				Required:     true,
			},
			{
				Key:          "quantity",
				Name:         "Order Quantity",
				Description:  "Fixed quantity per order",
				Type:         ParameterTypeDecimal,
				DefaultValue: decimal.NewFromFloat(0.01),
				MinValue:     decimal.NewFromFloat(0.001),
				MaxValue:     decimal.NewFromFloat(100),
				Required:     true,
			},
		},
		Indicators: []IndicatorConfig{
			{
				Type:       IndicatorTypeSMA,
				Parameters: map[string]interface{}{"period": "{{fast_period}}"},
				OutputKey:  "sma_fast",
			},
			{
				Type:       IndicatorTypeSMA,
				Parameters: map[string]interface{}{"period": "{{slow_period}}"},
				OutputKey:  "sma_slow",
			},
		},
		Rules: []RuleConfig{
			{
				Type: RuleTypeEntry,
				Condition: Condition{
					Operator: OperatorCrossAbove,
					Left:     ValueSource{Type: ValueSourceIndicator, Value: "sma_fast"},
					Right:    ValueSource{Type: ValueSourceIndicator, Value: "sma_slow"},
				},
				Action: ActionConfig{
					Type: ActionTypeMarketOrder,
					Side: "BUY",
					Quantity: QuantityConfig{
						Type:  QuantityTypeFixed,
						Value: "{{quantity}}",
					},
					PriceType: PriceTypeMarket,
				},
			},
			{
				Type: RuleTypeExit,
				Condition: Condition{
					Operator: OperatorCrossBelow,
					Left:     ValueSource{Type: ValueSourceIndicator, Value: "sma_fast"},
					Right:    ValueSource{Type: ValueSourceIndicator, Value: "sma_slow"},
				},
				Action: ActionConfig{
					Type: ActionTypeClosePosition,
				},
			},
		},
	}

	// RSI Mean Reversion Strategy
	tl.templates["rsi_mean_revert"] = &StrategyTemplate{
		ID:          "rsi_mean_revert",
		Name:        "RSI Mean Reversion",
		Description: "Buy when RSI is oversold, sell when overbought",
		Category:    CategoryMeanRevert,
		Parameters: []ParameterDefinition{
			{
				Key:          "rsi_period",
				Name:         "RSI Period",
				Description:  "Period for RSI calculation",
				Type:         ParameterTypeInt,
				DefaultValue: 14,
				MinValue:     7,
				MaxValue:     30,
				Required:     true,
			},
			{
				Key:          "oversold_level",
				Name:         "Oversold Level",
				Description:  "RSI level considered oversold",
				Type:         ParameterTypeInt,
				DefaultValue: 30,
				MinValue:     10,
				MaxValue:     40,
				Required:     true,
			},
			{
				Key:          "overbought_level",
				Name:         "Overbought Level",
				Description:  "RSI level considered overbought",
				Type:         ParameterTypeInt,
				DefaultValue: 70,
				MinValue:     60,
				MaxValue:     90,
				Required:     true,
			},
			{
				Key:          "quantity",
				Name:         "Order Quantity",
				Description:  "Fixed quantity per order",
				Type:         ParameterTypeDecimal,
				DefaultValue: decimal.NewFromFloat(0.01),
				MinValue:     decimal.NewFromFloat(0.001),
				MaxValue:     decimal.NewFromFloat(100),
				Required:     true,
			},
		},
		Indicators: []IndicatorConfig{
			{
				Type:       IndicatorTypeRSI,
				Parameters: map[string]interface{}{"period": "{{rsi_period}}"},
				OutputKey:  "rsi",
			},
		},
		Rules: []RuleConfig{
			{
				Type: RuleTypeEntry,
				Condition: Condition{
					Operator: OperatorLessThan,
					Left:     ValueSource{Type: ValueSourceIndicator, Value: "rsi"},
					Right:    ValueSource{Type: ValueSourceStatic, Value: "{{oversold_level}}"},
				},
				Action: ActionConfig{
					Type: ActionTypeMarketOrder,
					Side: "BUY",
					Quantity: QuantityConfig{
						Type:  QuantityTypeFixed,
						Value: "{{quantity}}",
					},
					PriceType: PriceTypeMarket,
				},
			},
			{
				Type: RuleTypeExit,
				Condition: Condition{
					Operator: OperatorGreaterThan,
					Left:     ValueSource{Type: ValueSourceIndicator, Value: "rsi"},
					Right:    ValueSource{Type: ValueSourceStatic, Value: "{{overbought_level}}"},
				},
				Action: ActionConfig{
					Type: ActionTypeClosePosition,
				},
			},
		},
	}
}

// GetTemplate retrieves a template by ID.
func (tl *TemplateLibrary) GetTemplate(id string) (*StrategyTemplate, error) {
	template, exists := tl.templates[id]
	if !exists {
		return nil, fmt.Errorf("template %s not found", id)
	}
	return template, nil
}

// ListTemplates returns all available templates.
func (tl *TemplateLibrary) ListTemplates() []*StrategyTemplate {
	templates := make([]*StrategyTemplate, 0, len(tl.templates))
	for _, template := range tl.templates {
		templates = append(templates, template)
	}
	return templates
}

// AddTemplate adds a custom template to the library.
func (tl *TemplateLibrary) AddTemplate(template *StrategyTemplate) error {
	if template.ID == "" {
		return fmt.Errorf("template ID is required")
	}
	if _, exists := tl.templates[template.ID]; exists {
		return fmt.Errorf("template %s already exists", template.ID)
	}
	tl.templates[template.ID] = template
	return nil
}
