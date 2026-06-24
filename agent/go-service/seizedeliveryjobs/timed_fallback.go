package seizedeliveryjobs

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

const (
	timedFallbackDefaultPrimaryMinutes  = 30
	timedFallbackDefaultFallbackMinutes = 30
	timedFallbackConfigNode             = "SeizeDeliveryJobsTimedFallbackConfig"
)

type timedFallbackStage string

const (
	timedFallbackStagePrimary  timedFallbackStage = "Primary"
	timedFallbackStageFallback timedFallbackStage = "Fallback"
	timedFallbackStageStop     timedFallbackStage = "Stop"
)

type timedFallbackConfig struct {
	PrimaryTimeoutMinutes  int `json:"primary_timeout_minutes"`
	FallbackTimeoutMinutes int `json:"fallback_timeout_minutes"`
}

type timedFallbackRecognitionParam struct {
	Stage timedFallbackStage `json:"stage"`
}

var timedFallbackStartedAt = struct {
	sync.Mutex
	byTaskID map[int64]time.Time
}{
	byTaskID: make(map[int64]time.Time),
}

type SeizeDeliveryJobsTimedFallbackRecognition struct{}

type SeizeDeliveryJobsTimedFallbackResetAction struct{}

func (a *SeizeDeliveryJobsTimedFallbackResetAction) Run(_ *maa.Context, arg *maa.CustomActionArg) bool {
	if arg == nil {
		log.Error().
			Str("component", "SeizeDeliveryJobsTimedFallback").
			Msg("invalid reset action arg")
		return false
	}
	resetTimedFallbackStartTime(arg.TaskID, time.Now())
	return true
}

func (r *SeizeDeliveryJobsTimedFallbackRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	if ctx == nil || arg == nil {
		log.Error().
			Str("component", "SeizeDeliveryJobsTimedFallback").
			Msg("invalid recognition context")
		return nil, false
	}

	param, err := parseTimedFallbackRecognitionParam(arg.CustomRecognitionParam)
	if err != nil {
		log.Error().Err(err).
			Str("component", "SeizeDeliveryJobsTimedFallback").
			Msg("parse recognition param")
		return nil, false
	}

	cfg := loadTimedFallbackConfig(ctx)
	startedAt := timedFallbackStartTime(arg.TaskID, time.Now())
	elapsed := time.Since(startedAt)
	stage := calculateTimedFallbackStage(elapsed, cfg)
	if stage != param.Stage {
		return nil, false
	}

	detail, _ := json.Marshal(map[string]any{
		"stage":                    stage,
		"elapsed_seconds":          int(elapsed.Seconds()),
		"primary_timeout_minutes":  cfg.PrimaryTimeoutMinutes,
		"fallback_timeout_minutes": cfg.FallbackTimeoutMinutes,
	})

	return &maa.CustomRecognitionResult{
		Box:    arg.Roi,
		Detail: string(detail),
	}, true
}

func parseTimedFallbackRecognitionParam(raw string) (timedFallbackRecognitionParam, error) {
	var param timedFallbackRecognitionParam
	if strings.TrimSpace(raw) == "" {
		return param, fmt.Errorf("custom_recognition_param is required")
	}
	if err := json.Unmarshal([]byte(raw), &param); err != nil {
		return param, err
	}

	switch param.Stage {
	case timedFallbackStagePrimary, timedFallbackStageFallback, timedFallbackStageStop:
		return param, nil
	default:
		return param, fmt.Errorf("unsupported stage: %q", param.Stage)
	}
}

func loadTimedFallbackConfig(ctx *maa.Context) timedFallbackConfig {
	cfg := defaultTimedFallbackConfig()
	raw, err := ctx.GetNodeJSON(timedFallbackConfigNode)
	if err != nil {
		log.Warn().Err(err).
			Str("component", "SeizeDeliveryJobsTimedFallback").
			Str("node", timedFallbackConfigNode).
			Msg("load config node failed, using defaults")
		return cfg
	}

	var wrapper struct {
		Attach timedFallbackConfig `json:"attach"`
	}
	if err := json.Unmarshal([]byte(raw), &wrapper); err != nil {
		log.Warn().Err(err).
			Str("component", "SeizeDeliveryJobsTimedFallback").
			Str("node", timedFallbackConfigNode).
			Msg("parse config node failed, using defaults")
		return cfg
	}
	return normalizeTimedFallbackConfig(wrapper.Attach)
}

func defaultTimedFallbackConfig() timedFallbackConfig {
	return timedFallbackConfig{
		PrimaryTimeoutMinutes:  timedFallbackDefaultPrimaryMinutes,
		FallbackTimeoutMinutes: timedFallbackDefaultFallbackMinutes,
	}
}

func normalizeTimedFallbackConfig(cfg timedFallbackConfig) timedFallbackConfig {
	if cfg.PrimaryTimeoutMinutes <= 0 {
		cfg.PrimaryTimeoutMinutes = timedFallbackDefaultPrimaryMinutes
	}
	if cfg.FallbackTimeoutMinutes <= 0 {
		cfg.FallbackTimeoutMinutes = timedFallbackDefaultFallbackMinutes
	}
	return cfg
}

func timedFallbackStartTime(taskID int64, now time.Time) time.Time {
	timedFallbackStartedAt.Lock()
	defer timedFallbackStartedAt.Unlock()

	if startedAt, ok := timedFallbackStartedAt.byTaskID[taskID]; ok {
		return startedAt
	}
	timedFallbackStartedAt.byTaskID[taskID] = now
	return now
}

func resetTimedFallbackStartTime(taskID int64, now time.Time) {
	timedFallbackStartedAt.Lock()
	defer timedFallbackStartedAt.Unlock()

	timedFallbackStartedAt.byTaskID[taskID] = now
}

func calculateTimedFallbackStage(elapsed time.Duration, cfg timedFallbackConfig) timedFallbackStage {
	cfg = normalizeTimedFallbackConfig(cfg)
	primaryTimeout := time.Duration(cfg.PrimaryTimeoutMinutes) * time.Minute
	fallbackTimeout := time.Duration(cfg.FallbackTimeoutMinutes) * time.Minute
	if elapsed < primaryTimeout {
		return timedFallbackStagePrimary
	}
	if elapsed < primaryTimeout+fallbackTimeout {
		return timedFallbackStageFallback
	}
	return timedFallbackStageStop
}

var (
	_ maa.CustomActionRunner      = &SeizeDeliveryJobsTimedFallbackResetAction{}
	_ maa.CustomRecognitionRunner = &SeizeDeliveryJobsTimedFallbackRecognition{}
)
