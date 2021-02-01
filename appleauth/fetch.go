package appleauth

import (
	"fmt"
	"net/http"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-steplib/steps-deploy-to-itunesconnect-deliver/devportalservice"
)

// Credentials contains either Apple ID or APIKey auth info
type Credentials struct {
	AppleID     *AppleID
	APIKey      *devportalservice.JWTConnection
	TestDevices []devportalservice.TestDevice
}

// AppleID contains Apple ID auth info
//
// Without 2FA:
//   Required: username, password
// With 2FA:
//   Required: username, password, appSpecificPassword
//			   session (Only for Fastlane, set as FASTLANE_SESSION)
//
// As Fastlane spaceship uses:
//  - iTMSTransporter: it requires Username + Password (or App-specific password with 2FA)
//  - TunesAPI: it requires Username + Password (+ 2FA session with 2FA)
type AppleID struct {
	Username, Password           string
	Session, AppSpecificPassword string
}

// MissingAuthConfigError is returned in case no usable Apple App Store Connect / Developer Portal authenticaion is found
type MissingAuthConfigError struct {
}

func (*MissingAuthConfigError) Error() string {
	return "Apple Service authentication not configured"
}

const notConnected = `Connected Apple Developer Portal Account not found.
Most likely because there is no Apple Developer Portal Account connected to the build, or the build is running locally.
Read more: https://devcenter.bitrise.io/getting-started/configuring-bitrise-steps-that-require-apple-developer-account-data/`

// Select return valid Apple ID or API Key based authentication data, from the provided Bitrise Apple Developer Connection or Inputs
// authSources: required, array of checked sources (in order, the first set one will be used)
//	 for example: []AppleAuthSource{&SourceConnectionAPIKey{}, &SourceConnectionAppleID{}, &SourceInputAPIKey{}, &SourceInputAppleID{}}
// inputs: optional, user provided inputs that are not centrally managed (by setting up connections)
func Select(devportalConnectionProvider devportalservice.AppleDeveloperConnectionProvider, authSources []Source, inputs Inputs) (Credentials, error) {
	initializeConnection := false
	for _, source := range authSources {
		initializeConnection = initializeConnection || source.RequiresConnection()
	}

	var conn *devportalservice.AppleDeveloperConnection
	if initializeConnection && devportalConnectionProvider != nil {
		var err error
		conn, err = devportalConnectionProvider.GetAppleDeveloperConnection()
		if err != nil {
			handleSessionDataError(err)
		}

		if conn == nil || (conn.JWTConnection == nil && conn.SessionConnection == nil) {
			fmt.Println()
			log.Debugf("%s", notConnected)
		}
	}

	for _, source := range authSources {
		auth, err := source.Fetch(conn, inputs)
		if err != nil {
			return Credentials{}, err
		}

		if auth != nil {
			fmt.Println()
			log.Infof("%s", source.Description())

			return *auth, nil
		}
	}

	return Credentials{}, &MissingAuthConfigError{}
}

func handleSessionDataError(err error) {
	if err == nil {
		return
	}

	if networkErr, ok := err.(devportalservice.NetworkError); ok && networkErr.Status == http.StatusNotFound {
		log.Debugf("")
		log.Debugf("%s", notConnected)
	} else {
		fmt.Println()
		log.Errorf("Failed to activate Bitrise Apple Developer Portal connection: %s", err)
		log.Warnf("Read more: https://devcenter.bitrise.io/getting-started/configuring-bitrise-steps-that-require-apple-developer-account-data/")
	}
}
