//go:build gorm

package hided

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GormString string

// GormValue implements gorm.Valuer to safely pass string data to gorm,
// you need to implements gorm.ParamsFilter to keep value secret into your lo
func (s String) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	return clause.Expr{
		SQL:  "?",
		Vars: []interface{}{GormString(s)},
	}
}

// String is the Stringer, it returns a clear string representation
func (s GormString) String() string {
	return string(s)
}

// Hiding is to return an obfuscated string
func (s GormString) Hiding() string {
	return "***"
}
