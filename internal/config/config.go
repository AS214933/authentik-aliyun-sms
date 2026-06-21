package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/example/authentik-aliyun-sms/internal/aliyun"
)

const (
	defaultHTTPAddr = ":8080"
	defaultEndpoint = "dysmsapi.aliyuncs.com"
)

type Config struct {
	HTTPAddr  string
	AuthToken string
	Aliyun    aliyun.Config
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr: strings.TrimSpace(getenv("HTTP_ADDR", defaultHTTPAddr)),
		Aliyun: aliyun.Config{
			AccessKeyID:     strings.TrimSpace(os.Getenv("ALIYUN_ACCESS_KEY_ID")),
			AccessKeySecret: strings.TrimSpace(os.Getenv("ALIYUN_ACCESS_KEY_SECRET")),
			Endpoint:        strings.TrimSpace(getenv("ALIYUN_ENDPOINT", defaultEndpoint)),
			Mode:            aliyun.Mode(strings.ToLower(strings.TrimSpace(getenv("ALIYUN_SMS_MODE", string(aliyun.ModeAuto))))),
			SignName:        strings.TrimSpace(os.Getenv("ALIYUN_SMS_SIGN_NAME")),
			TemplateCode:    strings.TrimSpace(os.Getenv("ALIYUN_SMS_TEMPLATE_CODE")),
			TemplateParam:   strings.TrimSpace(os.Getenv("ALIYUN_SMS_TEMPLATE_PARAM")),
			From:            strings.TrimSpace(os.Getenv("ALIYUN_SMS_FROM")),
			TimeoutSeconds:  getenvInt("ALIYUN_TIMEOUT_SECONDS", 10),
		},
		AuthToken: strings.TrimSpace(os.Getenv("AUTH_TOKEN")),
	}

	if err := validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func validate(cfg Config) error {
	var errs []error
	if cfg.HTTPAddr == "" {
		errs = append(errs, errors.New("HTTP_ADDR must not be empty"))
	}
	if cfg.Aliyun.AccessKeyID == "" {
		errs = append(errs, errors.New("ALIYUN_ACCESS_KEY_ID is required"))
	}
	if cfg.Aliyun.AccessKeySecret == "" {
		errs = append(errs, errors.New("ALIYUN_ACCESS_KEY_SECRET is required"))
	}
	if cfg.Aliyun.Endpoint == "" {
		errs = append(errs, errors.New("ALIYUN_ENDPOINT must not be empty"))
	}
	if !cfg.Aliyun.Mode.Valid() {
		errs = append(errs, fmt.Errorf("ALIYUN_SMS_MODE must be one of %q, %q, or %q", aliyun.ModeAuto, aliyun.ModeMainland, aliyun.ModeGlobal))
	}
	if (cfg.Aliyun.Mode == aliyun.ModeAuto || cfg.Aliyun.Mode == aliyun.ModeMainland) && cfg.Aliyun.TemplateCode == "" {
		errs = append(errs, errors.New("ALIYUN_SMS_TEMPLATE_CODE is required for auto and mainland modes"))
	}
	if cfg.Aliyun.TemplateParam == "" && (cfg.Aliyun.Mode == aliyun.ModeAuto || cfg.Aliyun.Mode == aliyun.ModeMainland) {
		errs = append(errs, errors.New("ALIYUN_SMS_TEMPLATE_PARAM is required for auto and mainland modes"))
	}
	if cfg.Aliyun.TimeoutSeconds <= 0 {
		errs = append(errs, errors.New("ALIYUN_TIMEOUT_SECONDS must be greater than zero"))
	}
	return errors.Join(errs...)
}

func getenv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}
