//go:build gorm

/*
 * Copyright 2024 Adrien Kara
 *
 * This program is free software: you can redistribute it and/or modify
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
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package logging

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"gitlab.com/iglou.eu/goulc/hided"
)

type GormLogger struct {
	*slog.Logger
}

func NewGormLogger(log *slog.Logger) *GormLogger {
	return &GormLogger{
		log,
	}
}

func (_ *GormLogger) LogMode(_ logger.LogLevel) logger.Interface {
	return nil
}

func (g *GormLogger) Info(ctx context.Context, msg string, data ...any) {
	g.InfoContext(ctx, "Database", "message", msg, "data", data)
}

func (g *GormLogger) Warn(ctx context.Context, msg string, data ...any) {
	g.WarnContext(ctx, "Database", "message", msg, "data", data)
}

func (g *GormLogger) Error(ctx context.Context, msg string, data ...any) {
	g.ErrorContext(ctx, "Database", "message", msg, "data", data)
}

func (g *GormLogger) Trace(
	ctx context.Context, begin time.Time, fc func() (string, int64), err error,
) {
	sql, rows := fc()
	elapsed := time.Since(begin)

	msg := "Database trace"
	if err != nil {
		msg = err.Error()

		if !errors.Is(err, gorm.ErrRecordNotFound) {
			g.ErrorContext(ctx,
				msg, "elapsed", elapsed, "trace", sql, "rows affected", rows)
			return
		}
	}

	g.DebugContext(ctx,
		msg, "elapsed", elapsed, "trace", sql, "rows affected", rows)
}

// ParamsFilter implements gorm.ParamsFilter interface, it iterates through
// params and applies Hiding() when there is an hided.GormHider sensitive values
func (_ *GormLogger) ParamsFilter(
	_ context.Context, sql string, params ...any,
) (string, []any) {
	for i := range params {
		if sensitive, ok := params[i].(hided.GormHider); ok {
			params[i] = sensitive.Hiding()
		}
	}

	return sql, params
}
