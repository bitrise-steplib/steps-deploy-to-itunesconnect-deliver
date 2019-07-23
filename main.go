package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-steputils/tools"
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

// configs ...
type configs struct {
	IpaPath string `env:"ipa_path"`
	PkgPath string `env:"pkg_path"`

	ItunesconUser string          `env:"itunescon_user,required"`
	Password      stepconf.Secret `env:"password,required"`
	AppPassword   stepconf.Secret `env:"app_password"`

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

		if versionToInstall == "latest" {
			versionToInstall = ""
		}

		cmds, err := rubycommand.GemInstall(gemName, versionToInstall)
		if err != nil {
			return fmt.Errorf("failed to create command, error: %s", err)
		}

		for _, cmd := range cmds {
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

		newVersion := forceVersion
		if forceVersion == "latest" {
			newVersion = ""
		}

		if err := gemInstallWithRetry("fastlane", newVersion); err != nil {
			return nil, "", err
		}

		fastlaneCmdSlice := []string{"fastlane"}
		if newVersion != "" {
			fastlaneCmdSlice = append(fastlaneCmdSlice, fmt.Sprintf("_%s_", newVersion))
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

	log.Printf("Gemfile exist, checking fastlane version from Gemfile.lock")

	gemfileDir := filepath.Dir(gemfilePth)
	gemfileLockPth := filepath.Join(gemfileDir, "Gemfile.lock")

	bundleInstallCalled := false
	if exist, err := pathutil.IsPathExists(gemfileLockPth); err != nil {
		return nil, "", err
	} else if !exist {
		log.Printf("Gemfile.lock not exist at: %s, running 'bundle install' ...", gemfileLockPth)

		cmd := command.NewWithStandardOuts("bundle", "install").SetStdin(os.Stdin).SetDir(gemfileDir)
		if err := cmd.Run(); err != nil {
			return nil, "", err
		}

		bundleInstallCalled = true

		if exist, err := pathutil.IsPathExists(gemfileLockPth); err != nil {
			return nil, "", err
		} else if !exist {
			return nil, "", errors.New("gemfile.lock does not exist, even 'bundle install' was called")
		}
	}

	fastlane, err := gemVersionFromGemfileLock("fastlane", gemfileLockPth)
	if err != nil {
		return nil, "", err
	}

	if fastlane.Found {
		log.Printf("fastlane version defined in Gemfile.lock: %s, using bundler to call fastlane commands...", fastlane.Version)

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

			log.Donef("$ %s", installBundlerCommand.PrintableCommandArgs())
			fmt.Println()

			if err := installBundlerCommand.Run(); err != nil {
				return nil, "", fmt.Errorf("command failed, error: %s", err)
			}

			// install Gemfile.lock gems with `bundle [_version_] install ...`
			fmt.Println()
			log.Infof("Installing bundle")

			cmd, err := gems.BundleInstallCommand(bundlerVersion)
			if err != nil {
				return nil, "", fmt.Errorf("failed to create bundle command model, error: %s", err)
			}
			cmd.SetStdout(os.Stdout).SetStderr(os.Stderr)
			cmd.SetDir(gemfileDir)

			log.Donef("$ %s", cmd.PrintableCommandArgs())
			fmt.Println()

			if err := cmd.Run(); err != nil {
				return nil, "", fmt.Errorf("command failed, error: %s", err)
			}
		}

		return append(gems.BundleExecPrefix(bundlerVersion), "fastlane"), gemfileDir, nil
	}

	log.Printf("fastlane version not found in Gemfile.lock, using system installed fastlane...")

	return []string{"fastlane"}, "", nil
}

func main() {
	var cfg configs
	if err := stepconf.Parse(&cfg); err != nil {
		fail("Issue with input: %s", err)
	}

	stepconf.Print(cfg)

	//
	// Validate inputs
	if cfg.IpaPath == "" && cfg.PkgPath == "" {
		fail("Issue with input: no IpaPath nor PkgPath parameter specified")
	}

	if cfg.AppID == "" && cfg.BundleID == "" {
		fail("Issue with input: no AppID or BundleID parameter specified")
	}

	//
	// Fastlane session
	fs, err := devportalservice.SessionData()
	if err != nil {
		if networkErr, ok := err.(devportalservice.NetworkError); ok && networkErr.Status == http.StatusNotFound {
			log.Debugf("")
			log.Debugf("Connected Apple Developer Portal Account not found")
		} else {
			fmt.Println()
			log.Errorf("Failed to activate Bitrise Apple Developer Portal connection: %s", err)
			log.Warnf("Read more: https://devcenter.bitrise.io/getting-started/connecting-apple-dev-account/")
		}
	} else {
		fmt.Println()
		log.Infof("Connected Apple Developer Portal Account found, exposing FASTLANE_SESSION env var")

		if err := tools.ExportEnvironmentWithEnvman("FASTLANE_SESSION", fs); err != nil {
			fail("Failed to export FASTLANE_SESSION, error: %s", err)
		}

		if err := os.Setenv("FASTLANE_SESSION", fs); err != nil {
			fail("Failed to set FASTLANE_SESSION env, error: %s", err)
		}

		log.Donef("Session exported")
	}

	//
	// Setup
	fmt.Println()
	log.Infof("Setup")

	startTime := time.Now()

	fastlaneCmdSlice, workDir, err := ensureFastlaneVersionAndCreateCmdSlice(cfg.FastlaneVersion, cfg.GemfilePath)
	if err != nil {
		fail("Failed to ensure fastlane version, error: %s", err)
	}

	versionCmdSlice := append(fastlaneCmdSlice, "-v")
	versionCmd := command.NewWithStandardOuts(versionCmdSlice[0], versionCmdSlice[1:]...)
	log.Printf("$ %s", versionCmd.PrintableCommandArgs())
	if err := versionCmd.Run(); err != nil {
		fail("Failed to print fastlane version, error: %s", err)
	}

	elapsed := time.Since(startTime)

	log.Printf("Setup took %f seconds to complete", elapsed.Seconds())

	//
	// Main
	fmt.Println()
	log.Infof("Deploy")

	log.Printf(`**Note:** if your password
contains special characters
and you experience problems, please
consider changing your password
to something with only
alphanumeric characters.`)
	fmt.Println()

	log.Printf(`**Be advised**
that this step uses a well maintained, open source tool which
uses *undocumented and unsupported APIs* (because the current
iTunes Connect platform does not have a documented and supported API)
to perform the deployment.
This means that when the API changes
**this step might fail until the tool is updated**.`)
	fmt.Println()

	options := []string{}
	if cfg.Options != "" {
		opts, err := shellquote.Split(cfg.Options)
		if err != nil {
			fail("Failed to split options (%s), error: %s", cfg.Options, err)
		}
		options = opts
	}

	envs := []string{
		"DELIVER_PASSWORD=" + string(cfg.Password),
	}

	if string(cfg.AppPassword) != "" {
		envs = append(envs, "FASTLANE_APPLE_APPLICATION_SPECIFIC_PASSWORD="+string(cfg.AppPassword))
	}

	if cfg.ITMSParameters != "" {
		envs = append(envs, "DELIVER_ITMSTRANSPORTER_ADDITIONAL_UPLOAD_PARAMETERS="+cfg.ITMSParameters)
	}

	args := []string{
		"deliver",
		"--username", cfg.ItunesconUser,
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
	log.Donef("$ %s", cmd.PrintableCommandArgs())

	cmd.SetStdout(os.Stdout)
	cmd.SetStderr(os.Stderr)
	cmd.SetStdin(os.Stdin)
	cmd.AppendEnvs(envs...)
	if workDir != "" {
		cmd.SetDir(workDir)
	}

	fmt.Println()

	if err := cmd.Run(); err != nil {
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
