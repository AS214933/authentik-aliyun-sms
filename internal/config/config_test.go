package config

import (
	"testing"
)

func TestLoadRequiresTemplateForAutoMode(t *testing.T) {
	t.Setenv("ALIYUN_ACCESS_KEY_ID", "ak")
	t.Setenv("ALIYUN_ACCESS_KEY_SECRET", "secret")

	_, err := Load()
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestLoadGlobalModeDoesNotRequireTemplate(t *testing.T) {
	t.Setenv("ALIYUN_ACCESS_KEY_ID", "ak")
	t.Setenv("ALIYUN_ACCESS_KEY_SECRET", "secret")
	t.Setenv("ALIYUN_SMS_MODE", "global")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected valid config: %v", err)
	}
	if cfg.Aliyun.Mode != "global" {
		t.Fatalf("unexpected mode: %q", cfg.Aliyun.Mode)
	}
	if cfg.Aliyun.RegionID != "cn-hongkong" {
		t.Fatalf("unexpected region id: %q", cfg.Aliyun.RegionID)
	}
	if cfg.Aliyun.Endpoint != "dysmsapi-xman.cn-hongkong.aliyuncs.com" {
		t.Fatalf("unexpected endpoint: %q", cfg.Aliyun.Endpoint)
	}
}
