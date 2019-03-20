package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/command/rubycommand"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/retry"
	"github.com/bitrise-io/steps-deploy-to-itunesconnect-deliver/devportalservice"
	"github.com/bitrise-tools/go-steputils/stepconf"
	"github.com/bitrise-tools/go-steputils/tools"
	"github.com/kballard/go-shellquote"
)

// Configs ...
type Configs struct {
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
			return fmt.Errorf("Failed to create command, error: %s", err)
		}

		for _, cmd := range cmds {
			if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
				return fmt.Errorf("Gem install failed, output: %s, error: %s", out, err)
			}
		}

		return nil
	})
}

func gemVersionFromGemfileLockContent(gem, content string) string {
	relevantLines := []string{}
	lines := strings.Split(content, "\n")

	specsStart := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			break
		}

		if trimmed == "specs:" {
			specsStart = true
			continue
		}

		if specsStart {
			relevantLines = append(relevantLines, trimmed)
		}
	}

	exp := regexp.MustCompile(fmt.Sprintf(`^%s \((.+)\)`, gem))
	for _, line := range relevantLines {
		match := exp.FindStringSubmatch(line)
		if match != nil && len(match) == 2 {
			return match[1]
		}
	}

	return ""
}

func gemVersionFromGemfileLock(gem, gemfileLockPth string) (string, error) {
	content, err := fileutil.ReadStringFromFile(gemfileLockPth)
	if err != nil {
		return "", err
	}
	return gemVersionFromGemfileLockContent(gem, content), nil
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
			return nil, "", errors.New("Gemfile.lock does not exist, even 'bundle install' was called")
		}
	}

	fastlaneVersion, err := gemVersionFromGemfileLock("fastlane", gemfileLockPth)
	if err != nil {
		return nil, "", err
	}

	if fastlaneVersion != "" {
		log.Printf("fastlane version defined in Gemfile.lock: %s, using bundler to call fastlane commands...", fastlaneVersion)

		if !bundleInstallCalled {
			cmd := command.NewWithStandardOuts("bundle", "install").SetStdin(os.Stdin).SetDir(gemfileDir)
			if err := cmd.Run(); err != nil {
				return nil, "", err
			}
		}

		return []string{"bundle", "exec", "fastlane"}, gemfileDir, nil
	}

	log.Printf("fastlane version not found in Gemfile.lock, using system installed fastlane...")

	return []string{"fastlane"}, "", nil
}

func main() {
	var configs Configs
	if err := stepconf.Parse(&configs); err != nil {
		fail("Issue with input: %s", err)
	}

	stepconf.Print(configs)

	//
	// Validate inputs
	if configs.IpaPath == "" && configs.PkgPath == "" {
		fail("no IpaPath nor PkgPath parameter specified")
	}

	if configs.AppID == "" && configs.BundleID == "" {
		fail("no AppID or BundleID parameter specified")
	}

	//
	// Fastlane session
	fmt.Println()
	log.Infof("Ensure cookies for Apple Developer Portal")

	fs, errors := devportalservice.SessionData()
	if errors != nil {
		log.Warnf("Failed to activate the Bitrise Apple Developer Portal connection: %s\nRead more: https://devcenter.bitrise.io/getting-started/signing-up/connecting-apple-dev-account/\n\nerrors:")
		for _, err := range errors {
			log.Errorf("%s\n", err)
		}
	} else {
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

	fastlaneCmdSlice, workDir, err := ensureFastlaneVersionAndCreateCmdSlice(configs.FastlaneVersion, configs.GemfilePath)
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

	log.Printf(`**Be advised** log.Printf(that this
step uses a well maintained, open source tool which
uses *undocumented and unsupported APIs* (because the current
iTunes Connect platform does not have a documented and supported API)
to perform the deployment.
This means that when the API changes
**this step might fail until the tool is updated**.`)
	fmt.Println()

	options := []string{}
	if configs.Options != "" {
		opts, err := shellquote.Split(configs.Options)
		if err != nil {
			fail("Failed to split options (%s), error: %s", configs.Options, err)
		}
		options = opts
	}

	envs := []string{
		fmt.Sprintf("DELIVER_PASSWORD=%s", string(configs.Password)),
	}

	if string(configs.AppPassword) != "" {
		envs = append(envs, fmt.Sprintf("FASTLANE_APPLE_APPLICATION_SPECIFIC_PASSWORD=%s", string(configs.AppPassword)))
	}

	if configs.ITMSParameters != "" {
		envs = append(envs, fmt.Sprintf("DELIVER_ITMSTRANSPORTER_ADDITIONAL_UPLOAD_PARAMETERS=%s", configs.ITMSParameters))
	}

	args := []string{
		"deliver",
		"--username", configs.ItunesconUser,
	}

	if configs.AppID != "" {
		args = append(args, "--app", configs.AppID)

		//warn user if BundleID is also set
		if configs.BundleID != "" {
			log.Warnf("AppID parameter specified, BundleID will be ignored")
		}
	} else if configs.BundleID != "" {
		args = append(args, "--app_identifier", configs.BundleID)
	}

	if configs.TeamName != "" {
		args = append(args, "--team_name", configs.TeamName)

		//warn user if TeamID is also set
		if configs.TeamID != "" {
			log.Warnf("TeamName parameter specified, TeamID will be ignored")
		}
	} else if configs.TeamID != "" {
		args = append(args, "--team_id", configs.TeamID)
	}

	if configs.IpaPath != "" {
		tmpIpaPath, err := normalizeArtifactPath(configs.IpaPath)
		if err != nil {
			log.Warnf("failed to copy the %s to the temporarily dir, error: %s", filepath.Base(configs.IpaPath), err)
			tmpIpaPath = configs.IpaPath
		}
		args = append(args, "--ipa", tmpIpaPath)

	} else if configs.PkgPath != "" {
		tmpPkgPath, err := normalizeArtifactPath(configs.PkgPath)
		if err != nil {
			log.Warnf("failed to copy the %s to the temporarily dir, error: %s", filepath.Base(configs.PkgPath), err)
			tmpPkgPath = configs.PkgPath
		}
		args = append(args, "--pkg", tmpPkgPath)
	}

	if configs.SkipScreenshots == "yes" {
		args = append(args, "--skip_screenshots")
	}

	if configs.SkipMetadata == "yes" {
		args = append(args, "--skip_metadata")
	}

	if configs.SkipAppVersionUpdate == "yes" {
		args = append(args, "--skip_app_version_update")
	}

	args = append(args, "--force")

	if configs.SubmitForReview == "yes" {
		args = append(args, "--submit_for_review")
	}

	args = append(args, "--platform", configs.Platform)

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
	log.Printf("The app (.ipa) was successfully uploaded to [iTunes Connect](https://itunesconnect.apple.com), you should see it in the *Prerelease* section on the app's iTunes Connect page!")
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
