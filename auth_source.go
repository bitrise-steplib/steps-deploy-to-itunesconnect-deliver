package main

import (
	"fmt"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-steplib/steps-deploy-to-itunesconnect-deliver/devportalservice"
)

// AppleAuthSource returns a specific kind (Apple ID/API Key) Apple authentication data from a specific source (Bitrise Service, manual input)
type AppleAuthSource interface {
	Fetch(connection *devportalservice.AppleDeveloperConnection, inputs AppleAuthInputs) (*AppleAuth, error)
	Description() string
	RequiresConnection() bool
}

// ServiceAPIKey provides API Key from Bitrise Service
type ServiceAPIKey struct{}

// InputAPIKey provides API Key from manual input
type InputAPIKey struct{}

// ServiceAppleID provides Apple ID from Bitrise Service
type ServiceAppleID struct{}

// InputAppleID provides Apple ID from manual input
type InputAppleID struct{}

//
// ServiceAPIKey

// Description ...
func (*ServiceAPIKey) Description() string {
	return "Connected Apple Developer Portal Account for App Store Connect API found"
}

// RequiresConnection ...
func (*ServiceAPIKey) RequiresConnection() bool {
	return true
}

// Fetch ...
func (*ServiceAPIKey) Fetch(conn *devportalservice.AppleDeveloperConnection, inputs AppleAuthInputs) (*AppleAuth, error) {
	if conn == nil || conn.JWTConnection == nil { // Not configured
		return nil, nil
	}

	return &AppleAuth{
		APIKey: conn.JWTConnection,
	}, nil
}

//
// InputAPIKey

// Description ...
func (*InputAPIKey) Description() string {
	return "Authenticating using Step inputs (App Store Connect API)"
}

// RequiresConnection ...
func (*InputAPIKey) RequiresConnection() bool {
	return false
}

// Fetch ...
func (*InputAPIKey) Fetch(conn *devportalservice.AppleDeveloperConnection, inputs AppleAuthInputs) (*AppleAuth, error) {
	if inputs.APIKeyPath == "" { // Not configured
		return nil, nil
	}

	privateKey, keyID, err := fetchPrivateKey(inputs.APIKeyPath)
	if err != nil {
		return nil, fmt.Errorf("could not fetch private key (%s) specified as input: %v", inputs.APIKeyPath, err)
	}
	if len(privateKey) == 0 {
		return nil, fmt.Errorf("private key (%s) is empty", inputs.APIKeyPath)
	}

	return &AppleAuth{
		APIKey: &devportalservice.JWTConnection{
			IssuerID:   inputs.APIIssuer,
			KeyID:      keyID,
			PrivateKey: string(privateKey),
		},
	}, nil
}

//
// ServiceAppleID

// Description ...
func (*ServiceAppleID) Description() string {
	return "Connected session-based Apple Developer Portal Account found"
}

// RequiresConnection ...
func (*ServiceAppleID) RequiresConnection() bool {
	return true
}

// Fetch ...
func (*ServiceAppleID) Fetch(conn *devportalservice.AppleDeveloperConnection, inputs AppleAuthInputs) (*AppleAuth, error) {
	if conn == nil || conn.SessionConnection == nil { // No Apple ID configured
		return nil, nil
	}

	sessionConn := conn.SessionConnection
	if sessionConn.AppleID != inputs.Username {
		log.Warnf("Connected Apple Developer (%s) and App Store login account (%s) do not match.", sessionConn.AppleID, inputs.Username)
		return nil, nil
	}
	if expiry := sessionConn.Expiry(); expiry != nil && sessionConn.Expired() {
		log.Warnf("TFA session expired on %s.", expiry.String())
		return nil, nil
	}
	session, err := sessionConn.FastlaneLoginSession()
	if err != nil {
		handleSessionDataError(err)
		return nil, nil
	}

	return &AppleAuth{
		AppleID: &AppleIDAuth{
			username:            inputs.Username,
			session:             session,
			appSpecificPassword: inputs.AppSpecificPassword,
		},
	}, nil
}

//
// InputAppleID

// Description ...
func (*InputAppleID) Description() string {
	return "Authenticating using Step inputs (session-based)"
}

// RequiresConnection ...
func (*InputAppleID) RequiresConnection() bool {
	return false
}

// Fetch ...
func (*InputAppleID) Fetch(conn *devportalservice.AppleDeveloperConnection, inputs AppleAuthInputs) (*AppleAuth, error) {
	if inputs.Username == "" { // Not configured
		return nil, nil
	}

	return &AppleAuth{
		AppleID: &AppleIDAuth{
			username:            inputs.Username,
			password:            inputs.Password,
			appSpecificPassword: inputs.AppSpecificPassword,
		},
	}, nil
}
