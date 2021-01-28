package appleauth

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

// SourceConnectionAPIKey provides API Key from Bitrise Service
type SourceConnectionAPIKey struct{}

// SourceInputAPIKey provides API Key from manual input
type SourceInputAPIKey struct{}

// SourceConnectionAppleID provides Apple ID from Bitrise Service
type SourceConnectionAppleID struct{}

// SourceInputAppleID provides Apple ID from manual input
type SourceInputAppleID struct{}

//
// ServiceAPIKey

// Description ...
func (*SourceConnectionAPIKey) Description() string {
	return "Connected Apple Developer Portal Account for App Store Connect API found"
}

// RequiresConnection ...
func (*SourceConnectionAPIKey) RequiresConnection() bool {
	return true
}

// Fetch ...
func (*SourceConnectionAPIKey) Fetch(conn *devportalservice.AppleDeveloperConnection, inputs AppleAuthInputs) (*AppleAuth, error) {
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
func (*SourceInputAPIKey) Description() string {
	return "Authenticating using Step inputs (App Store Connect API)"
}

// RequiresConnection ...
func (*SourceInputAPIKey) RequiresConnection() bool {
	return false
}

// Fetch ...
func (*SourceInputAPIKey) Fetch(conn *devportalservice.AppleDeveloperConnection, inputs AppleAuthInputs) (*AppleAuth, error) {
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
func (*SourceConnectionAppleID) Description() string {
	return "Connected session-based Apple Developer Portal Account found"
}

// RequiresConnection ...
func (*SourceConnectionAppleID) RequiresConnection() bool {
	return true
}

// Fetch ...
func (*SourceConnectionAppleID) Fetch(conn *devportalservice.AppleDeveloperConnection, inputs AppleAuthInputs) (*AppleAuth, error) {
	if conn == nil || conn.SessionConnection == nil { // No Apple ID configured
		return nil, nil
	}

	sessionConn := conn.SessionConnection
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
			Username:            conn.SessionConnection.AppleID,
			Password:            conn.SessionConnection.Password,
			Session:             session,
			AppSpecificPassword: inputs.AppSpecificPassword,
		},
	}, nil
}

//
// InputAppleID

// Description ...
func (*SourceInputAppleID) Description() string {
	return "Authenticating using Step inputs (session-based). This method does not support 2FA."
}

// RequiresConnection ...
func (*SourceInputAppleID) RequiresConnection() bool {
	return false
}

// Fetch ...
func (*SourceInputAppleID) Fetch(conn *devportalservice.AppleDeveloperConnection, inputs AppleAuthInputs) (*AppleAuth, error) {
	if inputs.Username == "" { // Not configured
		return nil, nil
	}

	return &AppleAuth{
		AppleID: &AppleIDAuth{
			Username:            inputs.Username,
			Password:            inputs.Password,
			AppSpecificPassword: inputs.AppSpecificPassword,
		},
	}, nil
}
