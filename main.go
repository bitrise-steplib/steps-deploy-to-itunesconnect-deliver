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
	"github.com/bitrise-tools/go-steputils/input"
	"github.com/kballard/go-shellquote"
)

// ConfigsModel ...
type ConfigsModel struct {
	IpaPath              string
	PkgPath              string

	ItunesconUser        string
	Password             string
	AppPassword          string

	AppID                string
	BundleID             string
	SubmitForReview      string
	SkipMetadata         string
	SkipScreenshots      string
	SkipAppVersionUpdate string
	TeamID               string
	TeamName             string
	Platform             string
	Options              string

	GemfilePath          string
	FastlaneVersion      string
	ITMSParameters       string
}

func createConfigsModelFromEnvs() ConfigsModel {
	return ConfigsModel{
		IpaPath:              os.Getenv("ipa_path"),
		PkgPath:              os.Getenv("pkg_path"),

		ItunesconUser:        os.Getenv("itunescon_user"),
		Password:             os.Getenv("password"),
		AppPassword:          os.Getenv("app_password"),

		AppID:                os.Getenv("app_id"),
		BundleID:             os.Getenv("bundle_id"),
		SubmitForReview:      os.Getenv("submit_for_review"),
		SkipMetadata:         os.Getenv("skip_metadata"),
		SkipScreenshots:      os.Getenv("skip_screenshots"),
		SkipAppVersionUpdate: os.Getenv("skip_app_version_update"),
		TeamID:               os.Getenv("team_id"),
		TeamName:             os.Getenv("team_name"),
		Platform:             os.Getenv("platform"),
		Options:              os.Getenv("options"),

		GemfilePath:          os.Getenv("gemfile_path"),
		FastlaneVersion:      os.Getenv("fastlane_version"),
		ITMSParameters:       os.Getenv("itms_upload_parameters"),
	}
}

func (configs ConfigsModel) print() {
	log.Infof("Configs:")

	log.Printf("- IpaPath: %s", configs.IpaPath)
	log.Printf("- PkgPath: %s", configs.PkgPath)

	log.Printf("- ItunesconUser: %s", configs.ItunesconUser)
	log.Printf("- Password: %s", input.SecureInput(configs.Password))
	log.Printf("- AppPassword: %s", input.SecureInput(configs.AppPassword))

	log.Printf("- AppID: %s", configs.AppID)
	log.Printf("- BundleID: %s", configs.BundleID)
	log.Printf("- SubmitForReview: %s", configs.SubmitForReview)
	log.Printf("- SkipMetadata: %s", configs.SkipMetadata)
	log.Printf("- SkipScreenshots: %s", configs.SkipScreenshots)
	log.Printf("- SkipAppVersionUpdate: %s", configs.SkipAppVersionUpdate)
	log.Printf("- TeamID: %s", configs.TeamID)
	log.Printf("- TeamName: %s", configs.TeamName)
	log.Printf("- Platform: %s", configs.Platform)
	log.Printf("- Options: %s", configs.Options)

	log.Printf("- GemfilePath: %s", configs.GemfilePath)
	log.Printf("- FastlaneVersion: %s", configs.FastlaneVersion)
	log.Printf("- ITMSParameters: %s", configs.ITMSParameters)
}

func (configs ConfigsModel) validate() error {
	if configs.IpaPath == "" && configs.PkgPath == "" {
		return errors.New("no IpaPath nor PkgPath parameter specified")
	}

	if configs.IpaPath != "" {
		if err := input.ValidateIfPathExists(configs.IpaPath); err != nil {
			return fmt.Errorf("IpaPath %s", err)
		}
	}

	if configs.PkgPath != "" {
		if err := input.ValidateIfPathExists(configs.PkgPath); err != nil {
			return fmt.Errorf("PkgPath %s", err)
		}
	}

	if err := input.ValidateIfNotEmpty(configs.ItunesconUser); err != nil {
		return fmt.Errorf("ItunesconUser %s", err)
	}

	if err := input.ValidateIfNotEmpty(configs.Password); err != nil {
		return fmt.Errorf("Password %s", err)
	}

	if configs.AppID == "" && configs.BundleID == "" {
		return errors.New("no AppID or BundleID parameter specified")
	}

	if err := input.ValidateWithOptions(configs.SubmitForReview, "yes", "no"); err != nil {
		return fmt.Errorf("SubmitForReview, %s", err)
	}

	if err := input.ValidateWithOptions(configs.SkipMetadata, "yes", "no"); err != nil {
		return fmt.Errorf("SkipMetadata, %s", err)
	}

	if err := input.ValidateWithOptions(configs.SkipScreenshots, "yes", "no"); err != nil {
		return fmt.Errorf("SkipScreenshots, %s", err)
	}

	if err := input.ValidateWithOptions(configs.SkipAppVersionUpdate, "yes", "no"); err != nil {
		return fmt.Errorf("SkipAppVersionUpdate, %s", err)
	}

	if err := input.ValidateWithOptions(configs.Platform, "ios", "osx", "appletvos"); err != nil {
		return fmt.Errorf("Platform, %s", err)
	}

	return nil
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
	configs := createConfigsModelFromEnvs()

	fmt.Println()
	configs.print()

	if err := configs.validate(); err != nil {
		fail("Issue with input: %s", err)
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
		fmt.Sprintf("DELIVER_PASSWORD=%s", configs.Password),
	}

	if configs.AppPassword != "" {
		envs = append(envs, fmt.Sprintf("FASTLANE_APPLE_APPLICATION_SPECIFIC_PASSWORD=%s", configs.AppPassword))
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
