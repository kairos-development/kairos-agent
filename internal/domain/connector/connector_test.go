package connector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPermissions_IsOverPrivileged(t *testing.T) {
	tests := []struct {
		name  string
		perms Permissions
		want  bool
	}{
		{name: "none", perms: Permissions{}, want: false},
		{name: "withdraw", perms: Permissions{HasWithdraw: true}, want: true},
		{name: "transfer", perms: Permissions{HasTransfer: true}, want: true},
		{name: "both", perms: Permissions{HasWithdraw: true, HasTransfer: true}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.perms.IsOverPrivileged())
		})
	}
}

func TestPermissions_IsSufficientForTrading(t *testing.T) {
	tests := []struct {
		name  string
		perms Permissions
		want  bool
	}{
		{name: "read and trade", perms: Permissions{CanRead: true, CanTrade: true}, want: true},
		{name: "read only", perms: Permissions{CanRead: true}, want: false},
		{name: "trade only", perms: Permissions{CanTrade: true}, want: false},
		{name: "neither", perms: Permissions{}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.perms.IsSufficientForTrading())
		})
	}
}
