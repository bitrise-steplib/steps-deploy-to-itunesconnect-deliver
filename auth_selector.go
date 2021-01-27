package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-steplib/steps-deploy-to-itunesconnect-deliver/devportalservice"
)

type fastlaneAuth struct {
	AppleID *legacyFastlaneAuth
	JWT     *fastlaneJWTAuth
}

type legacyFastlaneAuth struct {
	session             string
	username, password  string
	appSpecificPassword string
}

type authInputs struct {
	// Apple ID (legacy)
	itunesConnectUser, itunesConnectPassword, appSpecificPassword string
	// API key (JWT)
	APIIssuer, APIKeyPath string
}

// func validateAuthParams(cfg authInputs) error {
// 	cfg.APIIssuer = strings.TrimSpace(cfg.APIIssuer)
// 	cfg.APIKeyPath = strings.TrimSpace(cfg.APIKeyPath)
// 	cfg.itunesConnectUser = strings.TrimSpace(cfg.itunesConnectUser)
// 	var (
// 		isJWTAuthType     = (cfg.APIKeyPath != "" || cfg.APIIssuer != "")
// 		isAppleIDAuthType = (cfg.appSpecificPassword != "" || cfg.itunesConnectUser != "" || cfg.itunesConnectPassword != "")
// 	)

// 	switch {

// 	case isAppleIDAuthType == isJWTAuthType:

// 		return fmt.Errorf("one type of authentication required, either provide itunescon_user with password and optionally app_password or api_key_path with api_issuer")

// 	case isAppleIDAuthType:

// 		if cfg.itunesConnectUser == "" {
// 			return fmt.Errorf("no itunescon_user provided")
// 		}
// 		if cfg.itunesConnectPassword == "" {
// 			return fmt.Errorf("no password provided")
// 		}

// 	case isJWTAuthType:

// 		if cfg.APIIssuer == "" {
// 			return fmt.Errorf("no api_issuer provided")
// 		}
// 		if cfg.APIKeyPath == "" {
// 			return fmt.Errorf("no api_key_path provided")
// 		}

// 	}

// 	return nil
// }

func authParams(connection bitriseAppleConnection, authConfig authInputs) (fastlaneAuth, error) {
	useConnection := connection == connectionAutomatic || connection == connectionAppleID || connection == connectionAPIKey
	useInputs := connection == connectionAutomatic || connection == connectionDisabled
	if useConnection { // Use Bitrise Apple Developer Connection
		fastlaneSession := ""
		buildURL, buildAPIToken := os.Getenv("BITRISE_BUILD_URL"), os.Getenv("BITRISE_BUILD_API_TOKEN")
		if buildURL != "" && buildAPIToken != "" {
			var provider devportalservice.AppleDeveloperConnectionProvider
			provider = devportalservice.NewBitriseClient(http.DefaultClient)

			conn, err := provider.GetAppleDeveloperConnection(buildURL, buildAPIToken)
			if err != nil {
				handleSessionDataError(err)
			}

			if conn != nil && conn.JWTConnection != nil {
				fmt.Println()
				log.Infof("Connected Apple Developer Portal Account using  App Store Connect API found")

				return fastlaneAuth{
					JWT: &fastlaneJWTAuth{
						KeyID:      conn.JWTConnection.KeyID,
						IssuerID:   conn.JWTConnection.IssuerID,
						PrivateKey: conn.JWTConnection.PrivateKey,
					},
				}, nil
			}
			if conn != nil && conn.SessionConnection != nil {
				fmt.Println()
				log.Infof("Connected session-based Apple Developer Portal Account found")

				sessionConn := conn.SessionConnection

				if sessionConn.AppleID != authConfig.itunesConnectUser {
					log.Warnf("Connected Apple Developer and App Store login account missmatch")
				} else if expiry := sessionConn.Expiry(); expiry != nil && sessionConn.Expired() {
					log.Warnf("TFA session expired on %s", expiry.String())
				} else if session, err := sessionConn.FastlaneLoginSession(); err != nil {
					handleSessionDataError(err)
				} else {
					fastlaneSession = session
				}
			}
		} else {
			log.Warnf("Step is not running on bitrise.io: BITRISE_BUILD_URL and BITRISE_BUILD_API_TOKEN envs are not set")
		}

		if fastlaneSession != "" {
			if authConfig.appSpecificPassword == "" {
				log.Warnf("Application-specific password input is required if using Apple ID (legacy) authentication.")
			}

			return fastlaneAuth{
				AppleID: &legacyFastlaneAuth{
					session:             fastlaneSession,
					appSpecificPassword: authConfig.appSpecificPassword,
				},
			}, nil
		}
	}

	if useInputs {
		if authConfig.APIKeyPath != "" {
			privateKey, keyID, err := fetchPrivateKey(authConfig.APIKeyPath)
			if err != nil {
				return fastlaneAuth{}, fmt.Errorf("could not fetch private key: %s", err)
			}
			if len(privateKey) == 0 {
				return fastlaneAuth{}, fmt.Errorf("private key is empty")
			}

			return fastlaneAuth{
				JWT: &fastlaneJWTAuth{
					IssuerID:   authConfig.APIIssuer,
					KeyID:      keyID,
					PrivateKey: string(privateKey),
				},
			}, nil
		}
		if authConfig.itunesConnectUser != "" {
			if authConfig.appSpecificPassword == "" {
				log.Warnf("Application-specific password input is required if using Apple ID (legacy) authentication.")
			}

			return fastlaneAuth{
				AppleID: &legacyFastlaneAuth{
					username:            authConfig.itunesConnectUser,
					password:            authConfig.itunesConnectPassword,
					appSpecificPassword: authConfig.appSpecificPassword,
				},
			}, nil
		}
	}

	return fastlaneAuth{}, fmt.Errorf("App Store Connect authentication is not configured.")
}
