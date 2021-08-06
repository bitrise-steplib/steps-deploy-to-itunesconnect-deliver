package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/bitrise-io/go-steputils/command/gems"
	"github.com/bitrise-io/go-steputils/command/rubycommand"
	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/retry"
	"github.com/bitrise-io/go-xcode/appleauth"
	"github.com/bitrise-io/go-xcode/devportalservice"
	"github.com/kballard/go-shellquote"
)

// Config ...
type Config struct {
	IpaPath string `env:"ipa_path"`
	PkgPath string `env:"pkg_path"`

	BitriseConnection string          `env:"connection,opt[automatic,api_key,apple_id,off]"`
	ItunesConnectUser string          `env:"itunescon_user"`
	Password          stepconf.Secret `env:"password"`
	AppPassword       stepconf.Secret `env:"app_password"`
	APIKeyPath        stepconf.Secret `env:"api_key_path"`
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

	// Used to get Bitrise Apple Developer Portal Connection
	BuildURL      string          `env:"BITRISE_BUILD_URL"`
	BuildAPIToken stepconf.Secret `env:"BITRISE_BUILD_API_TOKEN"`
}

const latestStable = "latest-stable"
const latestPrerelease = "latest"

func parseAuthSources(bitriseConnection string) ([]appleauth.Source, error) {
	switch bitriseConnection {
	case "automatic":
		return []appleauth.Source{
			&appleauth.ConnectionAPIKeySource{},
			&appleauth.ConnectionAppleIDFastlaneSource{},
			&appleauth.InputAPIKeySource{},
			&appleauth.InputAppleIDFastlaneSource{},
		}, nil
	case "api_key":
		return []appleauth.Source{&appleauth.ConnectionAPIKeySource{}}, nil
	case "apple_id":
		return []appleauth.Source{&appleauth.ConnectionAppleIDFastlaneSource{}}, nil
	case "off":
		return []appleauth.Source{
			&appleauth.InputAPIKeySource{},
			&appleauth.InputAppleIDFastlaneSource{},
		}, nil
	default:
		return nil, fmt.Errorf("invalid connection input: %s", bitriseConnection)
	}
}

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

func (cfg Config) validate() error {
	if cfg.IpaPath == "" && cfg.PkgPath == "" {
		return fmt.Errorf("no IpaPath nor PkgPath parameter specified")
	}

	if cfg.AppID == "" && cfg.BundleID == "" {
		return fmt.Errorf("no AppID or BundleID parameter specified")
	}

	return nil
}

const notConnected = `Connected Apple Developer Portal Account not found.
Most likely because there is no Apple Developer Portal Account connected to the build.
Read more: https://devcenter.bitrise.io/getting-started/configuring-bitrise-steps-that-require-apple-developer-account-data/`

func handleSessionDataError(err error) {
	if err == nil {
		return
	}

	if networkErr, ok := err.(devportalservice.NetworkError); ok && networkErr.Status == http.StatusUnauthorized {
		fmt.Println()
		log.Warnf("%s", "Unauthorized to query Connected Apple Developer Portal Account. This happens by design, with a public app's PR build, to protect secrets.")

		return
	}

	fmt.Println()
	log.Errorf("Failed to activate Bitrise Apple Developer Portal connection: %s", err)
	log.Warnf("Read more: https://devcenter.bitrise.io/getting-started/configuring-bitrise-steps-that-require-apple-developer-account-data/")
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
		fail("Issue with input: %s", err)
	}
	authInputs := appleauth.Inputs{
		Username:            cfg.ItunesConnectUser,
		Password:            string(cfg.Password),
		AppSpecificPassword: string(cfg.AppPassword),
		APIIssuer:           cfg.APIIssuer,
		APIKeyPath:          string(cfg.APIKeyPath),
	}
	if err := authInputs.Validate(); err != nil {
		fail("Issue with authentication related inputs: %v", err)
	}

	//
	// Select and fetch Apple authenication source
	authSources, err := parseAuthSources(cfg.BitriseConnection)
	if err != nil {
		fail("Invalid input: unexpected value for Bitrise Apple Developer Connection (%s)", cfg.BitriseConnection)
	}

	var devportalConnectionProvider *devportalservice.BitriseClient
	if cfg.BuildURL != "" && cfg.BuildAPIToken != "" {
		devportalConnectionProvider = devportalservice.NewBitriseClient(retry.NewHTTPClient().StandardClient(), cfg.BuildURL, string(cfg.BuildAPIToken))
	} else {
		fmt.Println()
		log.Warnf("Connected Apple Developer Portal Account not found. Step is not running on bitrise.io: BITRISE_BUILD_URL and BITRISE_BUILD_API_TOKEN envs are not set")
	}
	var conn *devportalservice.AppleDeveloperConnection
	if cfg.BitriseConnection != "off" && devportalConnectionProvider != nil {
		var err error
		conn, err = devportalConnectionProvider.GetAppleDeveloperConnection()
		if err != nil {
			handleSessionDataError(err)
		}

		if conn != nil && (conn.APIKeyConnection == nil && conn.AppleIDConnection == nil) {
			fmt.Println()
			log.Warnf("%s", notConnected)
		}
	}

	authConfig, err := appleauth.Select(conn, authSources, authInputs)
	if err != nil {
		fail("Could not configure Apple Service authentication: %v", err)
	}
	if authConfig.AppleID != nil && authConfig.AppleID.AppSpecificPassword == "" {
		log.Warnf("If 2FA enabled Apple ID is used, Application-specific password is required.")
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

	var options []string
	if cfg.Options != "" {
		opts, err := shellquote.Split(cfg.Options)
		if err != nil {
			fail("Failed to split options (%s), error: %s", cfg.Options, err)
		}
		options = opts
	}

	envs := []string{}
	if cfg.ITMSParameters != "" {
		envs = append(envs, "DELIVER_ITMSTRANSPORTER_ADDITIONAL_UPLOAD_PARAMETERS="+cfg.ITMSParameters)
	}

	args := []string{
		"deliver",
	}

	authParams, err := FastlaneAuthParams(authConfig)
	if err != nil {
		fail("Failed to set up Fastlane authentication paramteres: %v", err)
	}
	for envKey, envValue := range authParams.Envs {
		envs = append(envs, fmt.Sprintf("%s=%s", envKey, envValue))
	}
	for _, arg := range authParams.Args {
		args = append(args, []string{arg.Key, arg.Value}...)
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
