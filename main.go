package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/command/gems"
	"github.com/bitrise-io/go-utils/command/rubycommand"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/retry"
	"github.com/bitrise-steplib/steps-deploy-to-itunesconnect-deliver/devportalservice"
	"github.com/kballard/go-shellquote"
)

// Config ...
type Config struct {
	IpaPath string `env:"ipa_path"`
	PkgPath string `env:"pkg_path"`

	ItunesConnectUser string          `env:"itunescon_user"`
	Password          stepconf.Secret `env:"password"`
	AppPassword       stepconf.Secret `env:"app_password"`
	APIKeyPath        string          `env:"api_key_path"`
	APIIssuer         string          `env:"api_issuer"`

	AppID                string `env:"app_id"`
	BundleID             string `env:"bundle_id"`
	SubmitForReview      string `env:"submit_for_review,opt[yes,no]"`
	SkipMetadata         string `env:"skip_metadata,opt[yes,no]"`
	SkipScreenshots      string `env:"skip_screenshots,opt[yes,no]"`
	SkipAppVersionUpdate string `env:"skip_app_version_update,opt[yes,no]"`
	TeamID               string `env:"team_id"`
	TeamName             string `env:"team_name"`
	Platform             string `env:"platform,opt[ios,osx,appletvos]"`
	Options              string `env:"options"`

	GemfilePath     string `env:"gemfile_path"`
	FastlaneVersion string `env:"fastlane_version"`
	ITMSParameters  string `env:"itms_upload_parameters"`

	VerboseLog bool `env:"verbose_log,opt[yes,no]"`
}

const latestStable = "latest-stable"
const latestPrerelease = "latest"

func fail(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

func gemInstallWithRetry(gemName string, version string) error {
	return retry.Times(2).Try(func(attempt uint) error {
		if attempt > 0 {
			log.Warnf("%d attempt failed", attempt+1)
		}

		versionToInstall := version

		if version == latestStable ||
			version == latestPrerelease {
			versionToInstall = ""
		}

		cmds, err := rubycommand.GemInstall(gemName, versionToInstall, version == latestPrerelease)
		if err != nil {
			return fmt.Errorf("failed to create command, error: %s", err)
		}

		for _, cmd := range cmds {
			fmt.Println()
			log.Donef("$ %s", cmd.PrintableCommandArgs())
			if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
				return fmt.Errorf("gem install command failed, output: %s, error: %s", out, err)
			}
		}

		return nil
	})
}

func gemVersionFromGemfileLock(gem, gemfileLockPth string) (gems.Version, error) {
	content, err := fileutil.ReadStringFromFile(gemfileLockPth)
	if err != nil {
		return gems.Version{}, err
	}
	return gems.ParseVersionFromBundle(gem, content)
}

func ensureFastlaneVersionAndCreateCmdSlice(forceVersion, gemfilePth string) ([]string, string, error) {
	if forceVersion != "" {
		log.Printf("fastlane version defined: %s, installing...", forceVersion)

		if err := gemInstallWithRetry("fastlane", forceVersion); err != nil {
			return nil, "", err
		}

		fastlaneCmdSlice := []string{"fastlane"}
		if forceVersion != latestStable && forceVersion != latestPrerelease {
			fastlaneCmdSlice = append(fastlaneCmdSlice, fmt.Sprintf("_%s_", forceVersion))
		}

		return fastlaneCmdSlice, "", nil
	}

	if gemfilePth == "" {
		log.Printf("no fastlane version nor Gemfile path defined, using system installed fastlane...")
		return []string{"fastlane"}, "", nil
	}

	if exist, err := pathutil.IsPathExists(gemfilePth); err != nil {
		return nil, "", err
	} else if !exist {
		log.Printf("Gemfile not exist at: %s and no fastlane version defined, using system installed fastlane...", gemfilePth)
		return []string{"fastlane"}, "", nil
	}

	log.Printf("Gemfile exist, checking Fastlane version from gem lockfile")

	bundleInstallCalled := false
	gemfileDir := filepath.Dir(gemfilePth)
	gemfileLockPth, err := gems.GemFileLockPth(gemfileDir)
	if err != nil {
		if err == gems.ErrGemLockNotFound {
			log.Printf("gem lockfile not exist at: %s, running 'bundle install' ...", gemfileDir)

			cmd := command.NewWithStandardOuts("bundle", "install").SetStdin(os.Stdin).SetDir(gemfileDir)
			if err := cmd.Run(); err != nil {
				return nil, "", err
			}

			bundleInstallCalled = true

			gemfileLockPth, err = gems.GemFileLockPth(gemfileDir)
			if err != nil {
				if err == gems.ErrGemLockNotFound {
					return nil, "", errors.New("gem lockfile still not exist, even after 'bundle install' was called")
				}
				return nil, "", err
			}
		} else {
			return nil, "", err
		}
	}

	fastlane, err := gemVersionFromGemfileLock("fastlane", gemfileLockPth)
	if err != nil {
		return nil, "", err
	}

	if fastlane.Found {
		log.Printf("fastlane version defined in gem lockfile: %s, using bundler to call fastlane commands...", fastlane.Version)

		var bundlerVersion gems.Version
		if !bundleInstallCalled {
			content, err := fileutil.ReadStringFromFile(gemfileLockPth)
			if err != nil {
				return nil, "", fmt.Errorf("failed to read file (%s) contents, error: %s", gemfileLockPth, err)
			}

			bundlerVersion, err = gems.ParseBundlerVersion(content)
			if err != nil {
				return nil, "", fmt.Errorf("failed to parse bundler version, error: %s", err)
			}

			fmt.Println()
			log.Infof("Installing bundler")

			// install bundler with `gem install bundler [-v version]`
			// in some configurations, the command "bunder _1.2.3_" can return 'Command not found', installing bundler solves this
			installBundlerCommand := gems.InstallBundlerCommand(bundlerVersion)
			installBundlerCommand.SetStdout(os.Stdout).SetStderr(os.Stderr)
			installBundlerCommand.SetDir(gemfileDir)

			fmt.Println()
			log.Donef("$ %s", installBundlerCommand.PrintableCommandArgs())

			if err := installBundlerCommand.Run(); err != nil {
				return nil, "", fmt.Errorf("command failed, error: %s", err)
			}

			// install gem lockfile gems with `bundle [_version_] install ...`
			fmt.Println()
			log.Infof("Installing bundle")

			cmd, err := gems.BundleInstallCommand(bundlerVersion)
			if err != nil {
				return nil, "", fmt.Errorf("failed to create bundle command model, error: %s", err)
			}
			cmd.SetStdout(os.Stdout).SetStderr(os.Stderr)
			cmd.SetDir(gemfileDir)

			fmt.Println()
			log.Donef("$ %s", cmd.PrintableCommandArgs())

			if err := cmd.Run(); err != nil {
				return nil, "", fmt.Errorf("command failed, error: %s", err)
			}
		}

		return append(gems.BundleExecPrefix(bundlerVersion), "fastlane"), gemfileDir, nil
	}

	log.Printf("Fastlane version not found in gem lockfile, using system installed Fastlane...")

	return []string{"fastlane"}, "", nil
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

func (cfg Config) validate() error {
	if cfg.IpaPath == "" && cfg.PkgPath == "" {
		return fmt.Errorf("Issue with input: no IpaPath nor PkgPath parameter specified")
	}

	if cfg.AppID == "" && cfg.BundleID == "" {
		return fmt.Errorf("Issue with input: no AppID or BundleID parameter specified")
	}

	var (
		isJWTAuthType     = (cfg.APIKeyPath != "" || cfg.APIIssuer != "")
		isAppleIDAuthType = (cfg.AppPassword != "" || cfg.Password != "" || cfg.ItunesConnectUser != "")
	)

	switch {

	case isAppleIDAuthType == isJWTAuthType:

		return fmt.Errorf("one type of authentication required, either provide itunescon_user with password and optionally app_password or api_key_path with api_issuer")

	case isAppleIDAuthType:

		if cfg.ItunesConnectUser == "" {
			return fmt.Errorf("no itunescon_user provided")
		}
		if cfg.Password == "" {
			return fmt.Errorf("no password provided")
		}

	case isJWTAuthType:

		if cfg.APIIssuer == "" {
			return fmt.Errorf("no api_issuer provided")
		}
		if cfg.APIKeyPath == "" {
			return fmt.Errorf("no api_key_path provided")
		}

	}

	return nil
}

func copyOrDownloadFile(u *url.URL, pth string) error {
	if err := os.MkdirAll(filepath.Dir(pth), 0777); err != nil {
		return err
	}

	certFile, err := os.Create(pth)
	if err != nil {
		return err
	}
	defer func() {
		if err := certFile.Close(); err != nil {
			log.Errorf("Failed to close file, error: %s", err)
		}
	}()

	// if file -> copy
	if u.Scheme == "file" {
		b, err := ioutil.ReadFile(u.Path)
		if err != nil {
			return err
		}
		_, err = certFile.Write(b)
		return err
	}

	// otherwise download
	f, err := http.Get(u.String())
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Body.Close(); err != nil {
			log.Errorf("Failed to close file, error: %s", err)
		}
	}()

	_, err = io.Copy(certFile, f.Body)
	return err
}

func getKeyID(u *url.URL) string {
	var keyID = "Bitrise" // as default if no ID found in file name

	// get the ID of the key from the file
	if matches := regexp.MustCompile(`AuthKey_(.+)\.p8`).FindStringSubmatch(filepath.Base(u.Path)); len(matches) == 2 {
		keyID = matches[1]
	}

	return keyID
}

func getKeyPath(keyID string, keyPaths []string) (string, bool, error) {
	certName := fmt.Sprintf("AuthKey_%s.p8", keyID)

	for _, path := range keyPaths {
		certPath := filepath.Join(path, certName)

		switch exists, err := pathutil.IsPathExists(certPath); {
		case err != nil:
			return "", false, err
		case exists:
			return certPath, true, err
		}
	}

	return filepath.Join(keyPaths[0], certName), false, nil
}

// APIKey is used to serialize App Store Connect API Key into JSON for fastlane
// see: https://docs.fastlane.tools/app-store-connect-api/#using-fastlane-api-key-json-file
type APIKey struct {
	KeyID    string `json:"key_id"`
	IssuerID string `json:"issuer_id"`
	Key      string `json:"key"`
}

func prepareAPIKey(apiKeyPath string, apiIssuer string) (string, error) {
	// see these in the altool's man page
	var keyPaths = []string{
		filepath.Join(os.Getenv("HOME"), ".appstoreconnect/private_keys"),
		filepath.Join(os.Getenv("HOME"), ".private_keys"),
		filepath.Join(os.Getenv("HOME"), "private_keys"),
		"./private_keys",
	}

	fileURL, err := url.Parse(apiKeyPath)
	if err != nil {
		return "", err
	}

	keyID := getKeyID(fileURL)

	keyPath, keyExists, err := getKeyPath(keyID, keyPaths)
	if err != nil {
		return "", err
	}

	if !keyExists {
		if err := copyOrDownloadFile(fileURL, keyPath); err != nil {
			return "", err
		}
	}

	key, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return "", err
	}

	json, err := json.Marshal(APIKey{keyID, apiIssuer, string(key)})
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

func main() {
	var cfg Config
	if err := stepconf.Parse(&cfg); err != nil {
		fail("Issue with input: %s", err)
	}

	stepconf.Print(cfg)
	log.SetEnableDebugLog(cfg.VerboseLog)

	//
	// Validate inputs
	if err := cfg.validate(); err != nil {
		fail("Input error: %s", err)
	}

	//
	// Fastlane session
	fastlaneSession := ""
	buildURL, buildAPIToken := os.Getenv("BITRISE_BUILD_URL"), os.Getenv("BITRISE_BUILD_API_TOKEN")
	if buildURL != "" && buildAPIToken != "" {
		var provider devportalservice.AppleDeveloperConnectionProvider
		provider = devportalservice.NewBitriseClient(http.DefaultClient)

		conn, err := provider.GetAppleDeveloperConnection(buildURL, buildAPIToken)
		if err != nil {
			handleSessionDataError(err)
		}

		if conn != nil && conn.AppleID != "" {
			fmt.Println()
			log.Infof("Connected session-based Apple Developer Portal Account found")

			if conn.AppleID != cfg.ItunesConnectUser {
				log.Warnf("Connected Apple Developer and App Store login account missmatch")
			} else if expiry := conn.Expiry(); expiry != nil && conn.Expired() {
				log.Warnf("TFA session expired on %s", expiry.String())
			} else if session, err := conn.FastlaneLoginSession(); err != nil {
				handleSessionDataError(err)
			} else {
				fastlaneSession = session
			}
		}
	} else {
		log.Warnf("Step is not running on bitrise.io: BITRISE_BUILD_URL and BITRISE_BUILD_API_TOKEN envs are not set")
	}

	//
	// Setup
	fmt.Println()
	log.Infof("Setup")

	startTime := time.Now()

	fastlaneCmdSlice, workDir, err := ensureFastlaneVersionAndCreateCmdSlice(cfg.FastlaneVersion, cfg.GemfilePath)
	if err != nil {
		fail("Failed to ensure Fastlane version, error: %s", err)
	}

	versionCmdSlice := append(fastlaneCmdSlice, "-v")
	versionCmd := command.NewWithStandardOuts(versionCmdSlice[0], versionCmdSlice[1:]...)
	fmt.Println()
	log.Donef(fmt.Sprintf("$ %s", versionCmd.PrintableCommandArgs()))
	if err := versionCmd.Run(); err != nil {
		fail("Failed to print Fastlane version, error: %s", err)
	}

	elapsed := time.Since(startTime)

	log.Printf("Setup took %f seconds to complete", elapsed.Seconds())

	//
	// Main
	fmt.Println()
	log.Infof("Deploy")

    if cfg.Password != "" {
	    log.Printf(`**Note:** if your password
contains special characters
and you experience problems, please
consider changing your password
to something with only
alphanumeric characters.`)
	    fmt.Println()
    }

    if cfg.APIKeyPath == "" {
	    log.Printf(`**Be advised**
that this step uses a well maintained, open source tool which
uses *undocumented and unsupported APIs* (because the current
iTunes Connect platform does not have a documented and supported API)
to perform the deployment.
This means that when the API changes
**this step might fail until the tool is updated**.`)
	    fmt.Println()
    }

	var options []string
	if cfg.Options != "" {
		opts, err := shellquote.Split(cfg.Options)
		if err != nil {
			fail("Failed to split options (%s), error: %s", cfg.Options, err)
		}
		options = opts
	}

	envs := []string{}
	if cfg.Password != "" {
		envs = append(envs, "DELIVER_PASSWORD="+string(cfg.Password))
	}

	if fastlaneSession != "" {
		envs = append(envs, "FASTLANE_SESSION="+fastlaneSession)
	}

	if string(cfg.AppPassword) != "" {
		envs = append(envs, "FASTLANE_APPLE_APPLICATION_SPECIFIC_PASSWORD="+string(cfg.AppPassword))
	}

	if cfg.ITMSParameters != "" {
		envs = append(envs, "DELIVER_ITMSTRANSPORTER_ADDITIONAL_UPLOAD_PARAMETERS="+cfg.ITMSParameters)
	}

	args := []string{
		"deliver",
	}

	if cfg.ItunesConnectUser != "" {
		args = append(args, "--username", cfg.ItunesConnectUser)
	}

	if string(cfg.APIKeyPath) != "" {
		apiKey, err := prepareAPIKey(cfg.APIKeyPath, cfg.APIIssuer)
		if err != nil {
			fail("Failed to prepare api key json for authentication, error: %s", err)
		}
		args = append(args, "--api_key_path", apiKey)
		// deliver: "Precheck cannot check In-app purchases with the App Store Connect API Key (yet). Exclude In-app purchases from precheck"
		args = append(args, "--precheck_include_in_app_purchases", "false")
	}

	if cfg.AppID != "" {
		args = append(args, "--app", cfg.AppID)

		//warn user if BundleID is also set
		if cfg.BundleID != "" {
			log.Warnf("AppID parameter specified, BundleID will be ignored")
		}
	} else if cfg.BundleID != "" {
		args = append(args, "--app_identifier", cfg.BundleID)
	}

	if cfg.TeamName != "" {
		args = append(args, "--team_name", cfg.TeamName)

		//warn user if TeamID is also set
		if cfg.TeamID != "" {
			log.Warnf("TeamName parameter specified, TeamID will be ignored")
		}
	} else if cfg.TeamID != "" {
		args = append(args, "--team_id", cfg.TeamID)
	}

	if cfg.IpaPath != "" {
		tmpIpaPath, err := normalizeArtifactPath(cfg.IpaPath)
		if err != nil {
			log.Warnf("failed to copy the %s to the temporarily dir, error: %s", filepath.Base(cfg.IpaPath), err)
			tmpIpaPath = cfg.IpaPath
		}
		args = append(args, "--ipa", tmpIpaPath)

	} else if cfg.PkgPath != "" {
		tmpPkgPath, err := normalizeArtifactPath(cfg.PkgPath)
		if err != nil {
			log.Warnf("failed to copy the %s to the temporarily dir, error: %s", filepath.Base(cfg.PkgPath), err)
			tmpPkgPath = cfg.PkgPath
		}
		args = append(args, "--pkg", tmpPkgPath)
	}

	if cfg.SkipScreenshots == "yes" {
		args = append(args, "--skip_screenshots")
	}

	if cfg.SkipMetadata == "yes" {
		args = append(args, "--skip_metadata")
	}

	if cfg.SkipAppVersionUpdate == "yes" {
		args = append(args, "--skip_app_version_update")
	}

	args = append(args, "--force")

	if cfg.SubmitForReview == "yes" {
		args = append(args, "--submit_for_review")
	}

	args = append(args, "--platform", cfg.Platform)

	args = append(args, options...)

	cmdSlice := append(fastlaneCmdSlice, args...)

	cmd := command.New(cmdSlice[0], cmdSlice[1:]...)
	fmt.Println()
	log.Donef("$ %s", cmd.PrintableCommandArgs())

	cmd.SetStdout(os.Stdout)
	cmd.SetStderr(os.Stderr)
	cmd.SetStdin(os.Stdin)
	if err := os.Unsetenv("FASTLANE_PASSWORD"); err != nil {
		fail("Could not unset Fastlane password, reason: ", err)
	}
	cmd.AppendEnvs(envs...)
	if workDir != "" {
		cmd.SetDir(workDir)
	}

	fmt.Println()

	if err := cmd.Run(); err != nil {
		if cfg.FastlaneVersion != latestPrerelease {
			log.Warnf(fmt.Sprintf(`If you have issues, use the latest prerelease version of fastlane.
Set the fastlane version input to "%s" to enable prerelease versions.`, latestPrerelease))
		}
		fail("Deploy failed, error: %s", err)
	}

	log.Donef("Success")
	log.Printf("The app (.ipa) was successfully uploaded to [App Store Connect](https://appstoreconnect.apple.com), you should see it in the *Prerelease* section on the app's page!")
}

func normalizeArtifactPath(pth string) (string, error) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("ipaOrPkg")
	if err != nil {
		return "", err
	}

	tmpPath := filepath.Join(tmpDir, "tmp"+filepath.Ext(pth))
	if err := command.CopyFile(pth, tmpPath); err != nil {
		return "", err
	}

	return tmpPath, nil
}
