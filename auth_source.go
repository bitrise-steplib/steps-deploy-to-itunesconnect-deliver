package main

import (
	"fmt"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-steplib/steps-deploy-to-itunesconnect-deliver/devportalservice"
)

// AppleAuthSource returns a specific kind (Apple ID/API Key) Apple authentication data from a specific source (Bitrise Service, manual input)
type AppleAuthSource interface {
	Fetch(connection *devportalservice.AppleDeveloperConnection, inputs AppleAuthInputs) (*AppleAuth, error)
}

// ServiceAppleID provides Apple ID from Bitrise Service
type ServiceAppleID struct{}

// ServiceAPIKey provides API Key from Bitrise Service
type ServiceAPIKey struct{}

// InputAppleID provides Apple ID from manual input
type InputAppleID struct{}

// InputAPIKey provides API Key from manual input
type InputAPIKey struct{}

// Fetch ...
func (*ServiceAppleID) Fetch(conn *devportalservice.AppleDeveloperConnection, inputs AppleAuthInputs) (*AppleAuth, error) {
	if conn == nil || conn.SessionConnection == nil { // No Apple ID configured
		return nil, nil
	}

	fmt.Println()
	log.Infof("Connected session-based Apple Developer Portal Account found")

	sessionConn := conn.SessionConnection
	if sessionConn.AppleID != inputs.Username {
		log.Warnf("Connected Apple Developer and App Store login account missmatch")
		return nil, nil
	}
	if expiry := sessionConn.Expiry(); expiry != nil && sessionConn.Expired() {
		log.Warnf("TFA session expired on %s", expiry.String())
		return nil, nil
	}
	session, err := sessionConn.FastlaneLoginSession()
	if err != nil {
		handleSessionDataError(err)
		return nil, nil
	}

	if inputs.AppSpecificPassword == "" {
		log.Warnf("Application-specific password input is required if using Apple ID (legacy) authentication.")
	}
	return &AppleAuth{
		AppleID: &AppleIDAuth{
			session:             session,
			appSpecificPassword: inputs.AppSpecificPassword,
		},
	}, nil
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

// Fetch ...
func (*ServiceAPIKey) Fetch(conn *devportalservice.AppleDeveloperConnection, inputs AppleAuthInputs) (*AppleAuth, error) {
	if conn == nil || conn.JWTConnection == nil { // Not configured
		return nil, nil
	}

	fmt.Println()
	log.Infof("Connected Apple Developer Portal Account using  App Store Connect API found")

	return &AppleAuth{
		APIKey: conn.JWTConnection,
	}, nil
}

// Fetch ...
func (*InputAppleID) Fetch(conn *devportalservice.AppleDeveloperConnection, inputs AppleAuthInputs) (*AppleAuth, error) {
	if inputs.Username == "" { // Not configured
		return nil, nil
	}

	if inputs.AppSpecificPassword == "" {
		log.Warnf("Application-specific password input is required if using Apple ID (legacy) authentication.")
	}
	return &AppleAuth{
		AppleID: &AppleIDAuth{
			username:            inputs.Username,
			password:            inputs.Password,
			appSpecificPassword: inputs.AppSpecificPassword,
		},
	}, nil
}
