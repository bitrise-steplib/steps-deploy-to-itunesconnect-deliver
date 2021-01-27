package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/bitrise-io/go-utils/pathutil"
)

// fastlaneJWTAuth is used to serialize App Store Connect API Key into JSON for fastlane
// see: https://docs.fastlane.tools/app-store-connect-api/#using-fastlane-api-key-json-file
type fastlaneJWTAuth struct {
	KeyID      string `json:"key_id"`
	IssuerID   string `json:"issuer_id"`
	PrivateKey string `json:"key"`
}

func writeFastlaneAPIKeyToFile(authData fastlaneJWTAuth) (string, error) {
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
