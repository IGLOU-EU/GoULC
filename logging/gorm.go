//go:build gorm

/*
 * Copyright 2024 Adrien Kara
 *
 * This file is part of GoULC.
 *
 * This is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package logging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"gitlab.com/iglou.eu/goulc/hided"
)

// GormLogger is a GORM logger adapter that wraps slog.Logger to implement
// gorm's logger.Interface. It formats database messages through the structured
// logging pipeline. The slog.Logger is kept unexported so GORM's Info, Warn,
// and Error signatures do not clash with promoted slog methods.
type GormLogger struct {
	log *slog.Logger
}

var (
	_ logger.Interface  = (*GormLogger)(nil)
	_ gorm.ParamsFilter = (*GormLogger)(nil)
)

// NewGormLogger creates a new GormLogger from an existing slog.Logger.
func NewGormLogger(log *slog.Logger) *GormLogger {
	return &GormLogger{
		log: log,
	}
}

// LogMode implements logger.Interface but is a no-op; log level is controlled
// by the underlying slog.Handler.
func (g *GormLogger) LogMode(_ logger.LogLevel) logger.Interface {
	return g
}

// Info logs a database info message, formatting the message and data with
// fmt.Sprintf.
func (g *GormLogger) Info(ctx context.Context, msg string, data ...any) {
	g.log.InfoContext(ctx, "Database", "message", fmt.Sprintf(msg, data...))
}

// Warn logs a database warning message, formatting the message and data with
// fmt.Sprintf.
func (g *GormLogger) Warn(ctx context.Context, msg string, data ...any) {
	g.log.WarnContext(ctx, "Database", "message", fmt.Sprintf(msg, data...))
}

// Error logs a database error message, formatting the message and data with
// fmt.Sprintf.
func (g *GormLogger) Error(ctx context.Context, msg string, data ...any) {
	g.log.ErrorContext(ctx, "Database", "message", fmt.Sprintf(msg, data...))
}

// Trace logs a database trace including elapsed time, SQL query, and rows
// affected. Errors other than gorm.ErrRecordNotFound are logged at ERROR level;
// everything else at DEBUG level.
func (g *GormLogger) Trace(
	ctx context.Context, begin time.Time, fc func() (string, int64), err error,
) {
	sql, rows := fc()
	elapsed := time.Since(begin)

	msg := "Database trace"
	if err != nil {
		msg = err.Error()

		if !errors.Is(err, gorm.ErrRecordNotFound) {
			g.log.ErrorContext(ctx,
				msg, "elapsed", elapsed, "trace", sql, "rows affected", rows)
			return
		}
	}

	g.log.DebugContext(ctx,
		msg, "elapsed", elapsed, "trace", sql, "rows affected", rows)
}

// ParamsFilter implements gorm.ParamsFilter interface, it iterates through
// params and applies Hiding() when there is an hided.GormHider sensitive values
func (*GormLogger) ParamsFilter(
	_ context.Context, sql string, params ...any,
) (string, []any) {
	// GORM may reuse the params slice for rebind or retry, so the
	// original values are copied instead of being masked in place.
	out := append([]any(nil), params...)
	for i := range out {
		if sensitive, ok := out[i].(hided.GormHider); ok {
			out[i] = sensitive.Hiding()
		}
	}

	return sql, out
}
