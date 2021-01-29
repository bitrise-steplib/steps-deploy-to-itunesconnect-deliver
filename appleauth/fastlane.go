package appleauth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/bitrise-io/go-utils/pathutil"
)

// fastlaneAPIKey is used to serialize App Store Connect API Key into JSON for fastlane
// see: https://docs.fastlane.tools/app-store-connect-api/#using-fastlane-api-key-json-file
type fastlaneAPIKey struct {
	KeyID      string `json:"key_id"`
	IssuerID   string `json:"issuer_id"`
	PrivateKey string `json:"key"`
}

type FastlaneParams struct {
	Envs, Args []string
}

func AppendFastlaneCredentials(p FastlaneParams, authConfig Credentials) error {
	if authConfig.AppleID != nil {
		// Set as environment variables
		if authConfig.AppleID.Password != "" {
			p.Envs = append(p.Envs, "DELIVER_PASSWORD="+authConfig.AppleID.Password)
		}

		if authConfig.AppleID.Session != "" {
			p.Envs = append(p.Envs, "FASTLANE_SESSION="+authConfig.AppleID.Session)
		}

		if authConfig.AppleID.AppSpecificPassword != "" {
			p.Envs = append(p.Envs, "FASTLANE_APPLE_APPLICATION_SPECIFIC_PASSWORD="+authConfig.AppleID.AppSpecificPassword)
		}

		// Add as an argument
		if authConfig.AppleID.Username != "" {
			p.Args = append(p.Args, "--username", authConfig.AppleID.Username)
		}
	}

	if authConfig.APIKey != nil {
		fastlaneAuthFile, err := writeFastlaneAPIKeyToFile(fastlaneAPIKey{
			IssuerID:   authConfig.APIKey.IssuerID,
			KeyID:      authConfig.APIKey.KeyID,
			PrivateKey: authConfig.APIKey.PrivateKey,
		})
		if err != nil {
			return fmt.Errorf("failed to write Fastane API Key configuration to file: %v", err)
		}

		p.Args = append(p.Args, "--api_key_path", fastlaneAuthFile)
		// deliver: "Precheck cannot check In-app purchases with the App Store Connect API Key (yet). Exclude In-app purchases from precheck"
		p.Args = append(p.Args, "--precheck_include_in_app_purchases", "false")
	}

	return nil
}

// writeFastlaneAPIKeyToFile writes a Fastlane-specific JSON file to disk, containing Apple Service authentication details
func writeFastlaneAPIKeyToFile(authData fastlaneAPIKey) (string, error) {
	json, err := json.Marshal(authData)
	if err != nil {
		return "", err
	}

	tmpDir, err := pathutil.NormalizedOSTempDirPath("apiKey")
	if err != nil {
		return "", err
	}
	tmpPath := filepath.Join(tmpDir, "api_key.json")

	if err := ioutil.WriteFile(tmpPath, json, os.ModePerm); err != nil {
		return "", err
	}

	return tmpPath, nil
}
