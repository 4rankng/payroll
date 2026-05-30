package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type TestConfig struct {
	BaseURL            string
	AdminUsername      string
	AdminPassword      string
	CommonPassword     string
	PartnerUsernames   []string
	FlexPayFixturePath string
}

func LoadTestConfig() *TestConfig {
	cfg := &TestConfig{
		BaseURL:            getEnvOrDefault("PAYROLL_BASE_URL", "http://localhost:8080"),
		AdminUsername:      getEnvOrDefault("PAYROLL_ADMIN_USER", "frankng"),
		AdminPassword:      getEnvOrDefault("PAYROLL_ADMIN_PASS", "Admin123"),
		CommonPassword:     getEnvOrDefault("PAYROLL_COMMON_PASS", "Admin123"),
		FlexPayFixturePath: getEnvOrDefault("PAYROLL_FLEXPAY_FIXTURE", "tests/fixtures/LGD- TINGTING 05.05.26 đợt 1.xlsx"),
	}

	partners := getEnvOrDefault("PAYROLL_PARTNER_USERS", "thanhmai,linhltk1,Vannth")
	cfg.PartnerUsernames = strings.Split(partners, ",")

	return cfg
}

func (c *TestConfig) UniquePrefix() string {
	return fmt.Sprintf("itest%s", time.Now().Format("20060102150405"))
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
