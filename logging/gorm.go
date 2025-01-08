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
)

type GormLogger struct {
	*slog.Logger
}

func NewGormLogger(logger *slog.Logger) *GormLogger {
	return &GormLogger{
		logger,
	}
}

func (g *GormLogger) LogMode(logger.LogLevel) logger.Interface {
	return nil
}

func (g *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	g.InfoContext(ctx, "Database", "message", msg, "data", data)
}

func (g *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	g.WarnContext(ctx, "Database", "message", msg, "data", data)
}

func (g *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	g.ErrorContext(ctx, "Database", "message", msg, "data", data)
}

func (g *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	sql, rows := fc()
	elapsed := time.Since(begin)

	msg := "Database trace"
	if err != nil {
		msg = err.Error()

		if !errors.Is(err, gorm.ErrRecordNotFound) {
			g.ErrorContext(ctx, msg, "elapsed", elapsed, "trace", sql, "rows affected", rows)
			return
		}
	}

	g.DebugContext(ctx, msg, "elapsed", elapsed, "trace", sql, "rows affected", rows)
}
