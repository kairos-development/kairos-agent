package entitlement

import "time"

type Edition string

const (
	EditionCommunity  Edition = "community"
	EditionPro        Edition = "pro"
	EditionTeam       Edition = "team"
	EditionEnterprise Edition = "enterprise"
)

type Feature string

const (
	FeatureMultiExchange   Feature = "multi_exchange"
	FeatureWebhookAlerts   Feature = "webhook_alerts"
	FeatureCloudSync       Feature = "cloud_sync"
	FeatureStrategyShare   Feature = "strategy_share"
	FeaturePrioritySupport Feature = "priority_support"
	FeatureAuditLog        Feature = "audit_log"
	FeatureSSO             Feature = "sso"
	FeatureCustomPlugins   Feature = "custom_plugins"
)

type Policy struct {
	Edition         Edition
	MaxExchanges    int
	MaxStrategies   int
	MaxConnectors   int
	AllowedFeatures []Feature
}

var policies = map[Edition]Policy{
	EditionCommunity: {
		Edition:         EditionCommunity,
		MaxExchanges:    1,
		MaxStrategies:   5,
		MaxConnectors:   2,
		AllowedFeatures: []Feature{},
	},
	EditionPro: {
		Edition:         EditionPro,
		MaxExchanges:    3,
		MaxStrategies:   50,
		MaxConnectors:   10,
		AllowedFeatures: []Feature{FeatureWebhookAlerts, FeatureCloudSync},
	},
	EditionTeam: {
		Edition:         EditionTeam,
		MaxExchanges:    10,
		MaxStrategies:   500,
		MaxConnectors:   50,
		AllowedFeatures: []Feature{FeatureWebhookAlerts, FeatureCloudSync, FeatureStrategyShare, FeaturePrioritySupport, FeatureAuditLog},
	},
	EditionEnterprise: {
		Edition:         EditionEnterprise,
		MaxExchanges:    -1,
		MaxStrategies:   -1,
		MaxConnectors:   -1,
		AllowedFeatures: []Feature{FeatureWebhookAlerts, FeatureCloudSync, FeatureStrategyShare, FeaturePrioritySupport, FeatureAuditLog, FeatureSSO, FeatureCustomPlugins},
	},
}

type Entitlement struct {
	Edition    Edition
	State      string
	ExpiresAt  time.Time
	GraceUntil time.Time
	Features   []Feature
	Limits     Limits
}

type Limits struct {
	MaxExchanges  int
	MaxStrategies int
	MaxConnectors int
}

func (e Edition) Policy() Policy {
	if p, ok := policies[e]; ok {
		return p
	}
	return policies[EditionCommunity]
}

func (p Policy) HasFeature(f Feature) bool {
	for _, af := range p.AllowedFeatures {
		if af == f {
			return true
		}
	}
	return false
}

func (p Policy) Unlimited(kind string) bool {
	switch kind {
	case "exchanges":
		return p.MaxExchanges == -1
	case "strategies":
		return p.MaxStrategies == -1
	case "connectors":
		return p.MaxConnectors == -1
	}
	return false
}
