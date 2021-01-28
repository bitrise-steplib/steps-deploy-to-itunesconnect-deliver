package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-steplib/steps-deploy-to-itunesconnect-deliver/devportalservice"
)

// AppleAuth contains either Apple ID or APIKey auth info
type AppleAuth struct {
	AppleID *AppleIDAuth
	APIKey  *devportalservice.JWTConnection
}

// AppleIDAuth contains Apple ID auth info
//
// Without 2FA:
//   Required: username, password
// With 2FA:
//   Required: username, password, session, appSpecificPassword
//
// As Fastlane spaceship uses:
//  - iTMSTransporter: it requires apple id + password (or appSpecificPassword in case of TFA)
//  - TunesAPI: it requires apple id + password (+ TFA session in case of TFA)
type AppleIDAuth struct {
	username, password           string
	session, appSpecificPassword string
}

// MissingAuthConfigError is returned in case no usable Apple App Store Connect / Developer Portal authenticaion is found
type MissingAuthConfigError struct {
}

func (*MissingAuthConfigError) Error() string {
	return "Apple Service authentication not configured"
}

// FetchAppleAuthData return valid Apple ID or API Key based authentication data, from the provided Bitrise Service or manual inputs
func FetchAppleAuthData(authSources []AppleAuthSource, inputs AppleAuthInputs) (AppleAuth, error) {
	if err := inputs.Validate(); err != nil {
		return AppleAuth{}, fmt.Errorf("input configuration is invalid: %s", err)
	}

	initializeConnection := false
	for _, source := range authSources {
		initializeConnection = initializeConnection || source.RequiresConnection()
	}

	var conn *devportalservice.AppleDeveloperConnection
	if initializeConnection {
		buildURL, buildAPIToken := os.Getenv("BITRISE_BUILD_URL"), os.Getenv("BITRISE_BUILD_API_TOKEN")
		if buildURL != "" && buildAPIToken != "" {
			provider := devportalservice.NewBitriseClient(http.DefaultClient)

			var err error
			conn, err = provider.GetAppleDeveloperConnection(buildURL, buildAPIToken)
			if err != nil {
				handleSessionDataError(err)
			}
		} else {
			log.Warnf("Step is not running on bitrise.io: BITRISE_BUILD_URL and BITRISE_BUILD_API_TOKEN envs are not set")
		}
	}

	for _, source := range authSources {
		auth, err := source.Fetch(conn, inputs)
		if err != nil {
			return AppleAuth{}, err
		}

		if auth != nil {
			fmt.Println()
			log.Infof("%s", source.Description())

			return *auth, nil
		}
	}

	return AppleAuth{}, &MissingAuthConfigError{}
}

func handleSessionDataError(err error) {
	if err == nil {
		return
	}

	if networkErr, ok := err.(devportalservice.NetworkError); ok && networkErr.Status == http.StatusNotFound {
		log.Debugf("")
		log.Debugf("Connected Apple Developer Portal Account not found")
		log.Debugf("Most likely because there is no Apple Developer Portal Account connected to the build, or the build is running locally.")
		log.Debugf("Read more: https://devcenter.bitrise.io/getting-started/configuring-bitrise-steps-that-require-apple-developer-account-data/")
	} else {
		fmt.Println()
		log.Errorf("Failed to activate Bitrise Apple Developer Portal connection: %s", err)
		log.Warnf("Read more: https://devcenter.bitrise.io/getting-started/configuring-bitrise-steps-that-require-apple-developer-account-data/")
	}
}
