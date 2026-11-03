package service

import (
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

func TestDefaultSyncSettingsEnablesBothSchedules(t *testing.T) {
	settings := DefaultSyncSettings()

	if !settings.PlayersEnabled || !settings.StatisticsEnabled {
		t.Errorf("los defaults deberían habilitar ambos schedules: %+v", settings)
	}
	if settings.PlayersInterval != 24*time.Hour || settings.StatisticsInterval != 24*time.Hour {
		t.Errorf("los defaults deberían ser diarios: %+v", settings)
	}
	if settings.RunTimeout <= 0 {
		t.Errorf("el timeout por defecto debe ser positivo: %v", settings.RunTimeout)
	}
}

func TestDefaultSchedulesBuildsDailyEnabledSchedules(t *testing.T) {
	now := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	schedules := defaultSchedules(DefaultSyncSettings(), now)

	if len(schedules) != 2 {
		t.Fatalf("defaultSchedules devolvió %d, se esperaban 2", len(schedules))
	}

	byType := make(map[model.SyncType]model.SyncSchedule, len(schedules))
	for _, schedule := range schedules {
		byType[schedule.Type] = schedule
	}

	for _, typ := range []model.SyncType{model.SyncTypePlayers, model.SyncTypeStatistics} {
		schedule, ok := byType[typ]
		if !ok {
			t.Fatalf("falta el schedule %s", typ)
		}
		if !schedule.Enabled {
			t.Errorf("el schedule %s no está habilitado", typ)
		}
		if schedule.IntervalMinutes != 1440 {
			t.Errorf("el schedule %s tiene intervalo %d, se esperaba 1440", typ, schedule.IntervalMinutes)
		}
		if !schedule.NextRunAt.Equal(now.Add(24 * time.Hour)) {
			t.Errorf("nextRunAt de %s = %v", typ, schedule.NextRunAt)
		}
	}
}

func TestIntervalMinutesClampsToMinimum(t *testing.T) {
	cases := []struct {
		name string
		in   time.Duration
		want int
	}{
		{"cero", 0, model.MinSyncIntervalMinutes},
		{"menor al mínimo", time.Minute, model.MinSyncIntervalMinutes},
		{"exacto", 15 * time.Minute, 15},
		{"diario", 24 * time.Hour, 1440},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := intervalMinutes(c.in); got != c.want {
				t.Errorf("intervalMinutes(%v) = %d, se esperaba %d", c.in, got, c.want)
			}
		})
	}
}
