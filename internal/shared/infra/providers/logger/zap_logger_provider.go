package logger

import (
	"golang_boilerplate_module/internal/config"
	"golang_boilerplate_module/internal/shared/domain/providers"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	otellog "go.opentelemetry.io/otel/log/global"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ZapLoggerProvider implements providers.LoggerProvider using zap.SugaredLogger.
// It writes to two destinations simultaneously (tee):
//  1. stdout — JSON (production) or console (development) format
//  2. OTLP → OTel Collector → Loki, via the otelzap bridge
//
// The otelzap core reads the global OTel LoggerProvider lazily at each write,
// so it works correctly even when OTel is initialized after the logger.
type ZapLoggerProvider struct {
	logger *zap.SugaredLogger
}

func NewZapLoggerProvider(cfg *config.Config) (providers.LoggerProvider, error) {
	level, err := zapcore.ParseLevel(cfg.Logger.Level)
	if err != nil {
		level = zapcore.ErrorLevel
	}

	// ── Core 1: stdout ────────────────────────────────────────────────────────
	var zapCfg zap.Config
	if cfg.App.Env == "development" {
		zapCfg = zap.NewDevelopmentConfig()
	} else {
		zapCfg = zap.NewProductionConfig()
	}
	zapCfg.Level = zap.NewAtomicLevelAt(level)
	zapCfg.InitialFields = map[string]any{
		"service": cfg.App.ServiceName,
		"version": cfg.App.Version,
	}

	base, err := zapCfg.Build()
	if err != nil {
		return nil, err
	}

	// ── Core 2: OTLP → OTel Collector → Loki ─────────────────────────────────
	// otelzap.NewCore uses otellog.GetLoggerProvider() lazily on every record,
	// so it picks up the real provider even when OTel is initialized after.
	otelCore := otelzap.NewCore(
		cfg.App.ServiceName,
		otelzap.WithLoggerProvider(otellog.GetLoggerProvider()),
	)

	// ── Tee: write to both cores simultaneously ────────────────────────────────
	teeLogger := zap.New(
		zapcore.NewTee(base.Core(), otelCore),
		zap.WithCaller(true),
		zap.AddCallerSkip(1),
	).With(
		zap.String("service", cfg.App.ServiceName),
		zap.String("version", cfg.App.Version),
	)

	return &ZapLoggerProvider{logger: teeLogger.Sugar()}, nil
}

func (z *ZapLoggerProvider) Info(msg string, fields ...any) {
	z.logger.Infow(msg, fields...)
}

func (z *ZapLoggerProvider) Warn(msg string, fields ...any) {
	z.logger.Warnw(msg, fields...)
}

func (z *ZapLoggerProvider) Error(msg string, fields ...any) {
	z.logger.Errorw(msg, fields...)
}

func (z *ZapLoggerProvider) Debug(msg string, fields ...any) {
	z.logger.Debugw(msg, fields...)
}

func (z *ZapLoggerProvider) With(args ...any) providers.LoggerProvider {
	return &ZapLoggerProvider{logger: z.logger.With(args...)}
}

func (z *ZapLoggerProvider) Sync() error {
	return z.logger.Sync()
}
