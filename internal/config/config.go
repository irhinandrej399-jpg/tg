package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config — корневая конфигурация приложения.
type Config struct {
	Telegram TelegramConfig `mapstructure:"telegram"`
	Accounts []Account      `mapstructure:"accounts"`
	Groups   []GroupPair    `mapstructure:"groups"`
	Schedule ScheduleConfig `mapstructure:"schedule"`
	Humanize HumanizeConfig `mapstructure:"humanize"`
	Database DatabaseConfig `mapstructure:"database"`
	Log      LogConfig      `mapstructure:"log"`
}

// TelegramConfig — реквизиты MTProto приложения.
// api_id и api_hash берутся из переменных окружения TG_API_ID / TG_API_HASH.
type TelegramConfig struct {
	APIID   int    `mapstructure:"api_id"`
	APIHash string `mapstructure:"api_hash"`
}

// Account описывает один userbot-аккаунт.
type Account struct {
	// Phone используется только как human-readable идентификатор в логах.
	Phone       string `mapstructure:"phone"`
	SessionFile string `mapstructure:"session_file"`
}

// GroupPair — пара мерчант-группа ↔ саппорт-группа.
type GroupPair struct {
	MerchantChatID int64 `mapstructure:"merchant_chat_id"`
	SupportChatID  int64 `mapstructure:"support_chat_id"`
}

// ScheduleConfig задаёт время смены аккаунтов (Europe/Moscow).
// Если аккаунт один — расписание игнорируется.
type ScheduleConfig struct {
	// DayStart — час начала дневной смены (первый аккаунт), default 8.
	DayStart int `mapstructure:"day_start"`
	// NightStart — час начала ночной смены (второй аккаунт), default 20.
	NightStart int `mapstructure:"night_start"`
}

// HumanizeConfig — параметры «человечного» поведения.
type HumanizeConfig struct {
	// DelayMin / DelayMax — диапазон задержки перед отправкой (секунды).
	DelayMin int `mapstructure:"delay_min"`
	DelayMax int `mapstructure:"delay_max"`
}

// DelayRange возвращает min и max задержку как time.Duration.
func (h HumanizeConfig) DelayRange() (min, max time.Duration) {
	return time.Duration(h.DelayMin) * time.Second,
		time.Duration(h.DelayMax) * time.Second
}

// DatabaseConfig — параметры подключения к PostgreSQL.
// DSN берётся из переменной окружения DATABASE_URL.
type DatabaseConfig struct {
	DSN string `mapstructure:"dsn"`
}

// LogConfig — настройки логирования.
type LogConfig struct {
	// Level: debug | info | warn | error
	Level string `mapstructure:"level"`
	// Format: json | text
	Format string `mapstructure:"format"`
}

// Load читает конфиг из файла path и перекрывает значения переменными окружения.
//
// Переменные окружения (приоритет выше yaml):
//
//	TG_API_ID      → telegram.api_id
//	TG_API_HASH    → telegram.api_hash
//	DATABASE_URL   → database.dsn
func Load(path string) (*Config, error) {
	v := viper.New()

	// --- файл конфига ---
	v.SetConfigFile(path)

	// --- значения по умолчанию ---
	v.SetDefault("schedule.day_start", 8)
	v.SetDefault("schedule.night_start", 20)
	v.SetDefault("humanize.delay_min", 1)
	v.SetDefault("humanize.delay_max", 4)
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")

	// --- переменные окружения ---
	v.AutomaticEnv()
	_ = v.BindEnv("telegram.api_id", "TG_API_ID")
	_ = v.BindEnv("telegram.api_hash", "TG_API_HASH")
	_ = v.BindEnv("database.dsn", "DATABASE_URL")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("config: read file %q: %w", path, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("config: unmarshal: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config: validation: %w", err)
	}

	return &cfg, nil
}

// validate проверяет обязательные поля.
func (c *Config) validate() error {
	if c.Telegram.APIID == 0 {
		return fmt.Errorf("telegram.api_id is required (set TG_API_ID env)")
	}
	if c.Telegram.APIHash == "" {
		return fmt.Errorf("telegram.api_hash is required (set TG_API_HASH env)")
	}
	if c.Database.DSN == "" {
		return fmt.Errorf("database.dsn is required (set DATABASE_URL env)")
	}
	if len(c.Accounts) == 0 {
		return fmt.Errorf("at least one account must be configured")
	}
	if len(c.Accounts) > 2 {
		return fmt.Errorf("maximum 2 accounts supported, got %d", len(c.Accounts))
	}
	for i, acc := range c.Accounts {
		if acc.SessionFile == "" {
			return fmt.Errorf("accounts[%d].session_file is required", i)
		}
	}
	if len(c.Groups) == 0 {
		return fmt.Errorf("at least one group pair must be configured")
	}
	for i, g := range c.Groups {
		if g.MerchantChatID == 0 || g.SupportChatID == 0 {
			return fmt.Errorf("groups[%d]: both merchant_chat_id and support_chat_id are required", i)
		}
	}
	if c.Schedule.DayStart < 0 || c.Schedule.DayStart > 23 {
		return fmt.Errorf("schedule.day_start must be 0–23")
	}
	if c.Schedule.NightStart < 0 || c.Schedule.NightStart > 23 {
		return fmt.Errorf("schedule.night_start must be 0–23")
	}
	if c.Humanize.DelayMin < 0 || c.Humanize.DelayMax < c.Humanize.DelayMin {
		return fmt.Errorf("humanize: delay_min must be >= 0 and delay_max >= delay_min")
	}
	return nil
}
