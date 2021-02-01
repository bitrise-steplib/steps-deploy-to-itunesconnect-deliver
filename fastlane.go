package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-steplib/steps-deploy-to-itunesconnect-deliver/appleauth"
)

// fastlaneAPIKey is used to serialize App Store Connect API Key into JSON for fastlane
// see: https://docs.fastlane.tools/app-store-connect-api/#using-fastlane-api-key-json-file
type fastlaneAPIKey struct {
	KeyID      string `json:"key_id"`
	IssuerID   string `json:"issuer_id"`
	PrivateKey string `json:"key"`
}

// FastlaneParams are Fastlane command arguments and environment variables
type FastlaneParams struct {
	Envs, Args map[string]string
}

// FastlaneAuthParams converts Apple credentials to Fastlane env vars and arguments
func FastlaneAuthParams(authConfig appleauth.Credentials) (FastlaneParams, error) {
	envs := make(map[string]string)
	args := make(map[string]string)
	if authConfig.AppleID != nil {
		// Set as environment variables
		if authConfig.AppleID.Password != "" {
			envs["DELIVER_PASSWORD"] = authConfig.AppleID.Password
		}

		if authConfig.AppleID.Session != "" {
			envs["FASTLANE_SESSION"] = authConfig.AppleID.Session
		}

		if authConfig.AppleID.AppSpecificPassword != "" {
			envs["FASTLANE_APPLE_APPLICATION_SPECIFIC_PASSWORD"] = authConfig.AppleID.AppSpecificPassword
		}

		// Add as an argument
		if authConfig.AppleID.Username != "" {
			args["--username"] = authConfig.AppleID.Username
		}
	}

	if authConfig.APIKey != nil {
		privateKey, err := json.Marshal(fastlaneAPIKey{
			IssuerID:   authConfig.APIKey.IssuerID,
			KeyID:      authConfig.APIKey.KeyID,
			PrivateKey: authConfig.APIKey.PrivateKey,
		})
		if err != nil {
			return FastlaneParams{}, fmt.Errorf("failed to marshal Fastane API Key configuration: %v", err)
		}

		tmpDir, err := pathutil.NormalizedOSTempDirPath("apiKey")
		if err != nil {
			return FastlaneParams{}, err
		}
		fastlaneAuthFile := filepath.Join(tmpDir, "api_key.json")
		if err := ioutil.WriteFile(fastlaneAuthFile, privateKey, os.ModePerm); err != nil {
			return FastlaneParams{}, err
		}

		args["--api_key_path"] = fastlaneAuthFile
		// deliver: "Precheck cannot check In-app purchases with the App Store Connect API Key (yet). Exclude In-app purchases from precheck"
		args["--precheck_include_in_app_purchases"] = "false"
	}

	return FastlaneParams{Envs: envs, Args: args}, nil
}
