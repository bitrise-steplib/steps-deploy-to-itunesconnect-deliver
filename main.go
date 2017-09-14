package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/command/rubycommand"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/retry"
	"github.com/bitrise-tools/go-steputils/input"
	"github.com/kballard/go-shellquote"
)

// ConfigsModel ...
type ConfigsModel struct {
	IpaPath string
	PkgPath string

	ItunesconUser string
	Password      string

	AppID           string
	BundleID        string
	SubmitForBeta   string
	SkipMetadata    string
	SkipScreenshots string
	TeamID          string
	TeamName        string
	Options         string

	UpdateDeliver string
	Platform      string
}

func createConfigsModelFromEnvs() ConfigsModel {
	return ConfigsModel{
		IpaPath: os.Getenv("ipa_path"),
		PkgPath: os.Getenv("pkg_path"),

		ItunesconUser: os.Getenv("itunescon_user"),
		Password:      os.Getenv("password"),

		AppID:           os.Getenv("app_id"),
		BundleID:        os.Getenv("bundle_id"),
		SubmitForBeta:   os.Getenv("submit_for_beta"),
		SkipMetadata:    os.Getenv("skip_metadata"),
		SkipScreenshots: os.Getenv("skip_screenshots"),
		TeamID:          os.Getenv("team_id"),
		TeamName:        os.Getenv("team_name"),
		Options:         os.Getenv("options"),

		UpdateDeliver: os.Getenv("update_deliver"),
		Platform:      os.Getenv("platform"),
	}
}

func (configs ConfigsModel) print() {
	log.Infof("Configs:")

	log.Printf("- IpaPath: %s", configs.IpaPath)
	log.Printf("- PkgPath: %s", configs.PkgPath)

	log.Printf("- ItunesconUser: %s", configs.ItunesconUser)
	log.Printf("- Password: %s", input.SecureInput(configs.Password))

	log.Printf("- AppID: %s", configs.AppID)
	log.Printf("- BundleID: %s", configs.BundleID)
	log.Printf("- SubmitForBeta: %s", configs.SubmitForBeta)
	log.Printf("- SkipMetadata: %s", configs.SkipMetadata)
	log.Printf("- SkipScreenshots: %s", configs.SkipScreenshots)
	log.Printf("- TeamID: %s", configs.TeamID)
	log.Printf("- TeamName: %s", configs.TeamName)
	log.Printf("- Options: %s", configs.Options)

	log.Printf("- UpdateDeliver: %s", configs.UpdateDeliver)
	log.Printf("- Platform: %s", configs.Platform)
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

	if err := input.ValidateWithOptions(configs.SubmitForBeta, "yes", "no"); err != nil {
		return fmt.Errorf("SubmitForBeta, %s", err)
	}

	if err := input.ValidateWithOptions(configs.SkipMetadata, "yes", "no"); err != nil {
		return fmt.Errorf("SkipMetadata, %s", err)
	}

	if err := input.ValidateWithOptions(configs.SkipScreenshots, "yes", "no"); err != nil {
		return fmt.Errorf("SkipScreenshots, %s", err)
	}

	if err := input.ValidateWithOptions(configs.UpdateDeliver, "yes", "no"); err != nil {
		return fmt.Errorf("UpdateDeliver, %s", err)
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

func ensureGemInstalled(gemName string, isUpgrade bool) error {
	installed, err := rubycommand.IsGemInstalled(gemName, "")
	if err != nil {
		return fmt.Errorf("Failed to check if gem (%s) installed, error: %s", gemName, err)
	}

	if installed {
		log.Printf("%s already installed", gemName)

		if !isUpgrade {
			log.Printf("update %s disabled, setup finished...", gemName)
		} else {
			log.Printf("updating %s...", gemName)

			err := retry.Times(2).Try(func(attempt uint) error {
				if attempt > 0 {
					log.Warnf("%d attempt failed", attempt+1)
				}

				cmds, err := rubycommand.GemUpdate(gemName)
				if err != nil {
					return fmt.Errorf("Failed to create command, error: %s", err)
				}

				for _, cmd := range cmds {
					if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
						return fmt.Errorf("Gem update failed, output: %s, error: %s", out, err)
					}
				}

				return nil
			})

			return err
		}
	} else {
		log.Printf("%s NOT yet installed, attempting install...", gemName)

		err := retry.Times(2).Try(func(attempt uint) error {
			if attempt > 0 {
				log.Warnf("%d attempt failed", attempt+1)
			}

			cmds, err := rubycommand.GemInstall(gemName, "")
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

		return err
	}

	return nil
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

	isUpdateGems := !(configs.UpdateDeliver == "no")
	for _, aGemName := range []string{"fastlane"} {
		if err := ensureGemInstalled(aGemName, isUpdateGems); err != nil {
			fail("Failed to install '%s', error: %s", aGemName, err)
		}
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
		args = append(args, "--ipa", configs.IpaPath)
	} else if configs.PkgPath != "" {
		args = append(args, "--pkg", configs.PkgPath)
	}

	if configs.SkipScreenshots == "yes" {
		args = append(args, "--skip_screenshots")
	}

	if configs.SkipMetadata == "yes" {
		args = append(args, "--skip_metadata")
	}

	args = append(args, "--force")

	if configs.SubmitForBeta == "yes" {
		args = append(args, "--submit_for_review")
	}

	args = append(args, "--platform", configs.Platform)

	args = append(args, options...)

	cmd := command.New("fastlane", args...)
	log.Donef("$ %s", cmd.PrintableCommandArgs())

	cmd.SetStdout(os.Stdout)
	cmd.SetStderr(os.Stderr)
	cmd.SetStdin(os.Stdin)
	cmd.AppendEnvs(envs...)

	fmt.Println()

	if err := cmd.Run(); err != nil {
		fail("Deploy failed, error: %s", err)
	}

	log.Donef("Success")
	log.Printf("The app (.ipa) was successfully uploaded to [iTunes Connect](https://itunesconnect.apple.com), you should see it in the *Prerelease* section on the app's iTunes Connect page!")
	log.Printf("**Don't forget to enable** the **TestFlight Beta Testing** switch on iTunes Connect (on the *Prerelease* tab of the app) if this is a new version of the app!")
}
