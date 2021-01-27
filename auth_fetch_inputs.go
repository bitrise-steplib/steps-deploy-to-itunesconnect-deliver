package main

import (
	"fmt"
	"strings"
)

// AppleAuthInputs are Apple App Store Connect / Developer Portal authentication configuration provided by end user
type AppleAuthInputs struct {
	// Apple ID (legacy)
	Username, Password, AppSpecificPassword string
	// API key (JWT)
	APIIssuer, APIKeyPath string
}

// Validate trims extra spaces and checks input grouping
func (cfg *AppleAuthInputs) Validate() error {
	cfg.APIIssuer = strings.TrimSpace(cfg.APIIssuer)
	cfg.APIKeyPath = strings.TrimSpace(cfg.APIKeyPath)
	cfg.Username = strings.TrimSpace(cfg.Username)
	var (
		isAPIKeyAuthType  = (cfg.APIKeyPath != "" || cfg.APIIssuer != "")
		isAppleIDAuthType = (cfg.AppSpecificPassword != "" || cfg.Username != "" || cfg.Password != "")
	)

	switch {
	case isAppleIDAuthType == isAPIKeyAuthType:
		return fmt.Errorf("one type of authentication required, either provide itunescon_user with password and optionally app_password or api_key_path with api_issuer")

	case isAppleIDAuthType:
		if cfg.Username == "" {
			return fmt.Errorf("no itunescon_user provided")
		}
		if cfg.Password == "" {
			return fmt.Errorf("no password provided")
		}

	case isAPIKeyAuthType:
		if cfg.APIIssuer == "" {
			return fmt.Errorf("no api_issuer provided")
		}
		if cfg.APIKeyPath == "" {
			return fmt.Errorf("no api_key_path provided")
		}
	}

	return nil
}
