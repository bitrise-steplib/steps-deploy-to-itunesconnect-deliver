package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/bitrise-io/go-utils/cmdex"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/steps-deploy-to-itunesconnect-deliver/rubycmd"
	"github.com/kballard/go-shellquote"
)

// ConfigsModel ...
type ConfigsModel struct {
	IpaPath string
	PkgPath string

	ItunesconUser string
	Password      string

	AppID           string
	SubmitForBeta   string
	SkipMetadata    string
	SkipScreenshots string
	TeamID          string
	TeamName        string
	Options         string

	UpdateDeliver string
}

func createConfigsModelFromEnvs() ConfigsModel {
	return ConfigsModel{
		IpaPath: os.Getenv("ipa_path"),
		PkgPath: os.Getenv("pkg_path"),

		ItunesconUser: os.Getenv("itunescon_user"),
		Password:      os.Getenv("password"),

		AppID:           os.Getenv("app_id"),
		SubmitForBeta:   os.Getenv("submit_for_beta"),
		SkipMetadata:    os.Getenv("skip_metadata"),
		SkipScreenshots: os.Getenv("skip_screenshots"),
		TeamID:          os.Getenv("team_id"),
		TeamName:        os.Getenv("team_name"),
		Options:         os.Getenv("options"),

		UpdateDeliver: os.Getenv("update_deliver"),
	}
}

func (configs ConfigsModel) print() {
	log.Info("Configs:")

	log.Detail("- IpaPath: %s", configs.IpaPath)
	log.Detail("- PkgPath: %s", configs.PkgPath)

	log.Detail("- ItunesconUser: %s", configs.ItunesconUser)

	securePassword := ""
	if configs.Password != "" {
		securePassword = "***"
	}
	log.Detail("- Password: %s", securePassword)

	log.Detail("- AppID: %s", configs.AppID)
	log.Detail("- SubmitForBeta: %s", configs.SubmitForBeta)
	log.Detail("- SkipMetadata: %s", configs.SkipMetadata)
	log.Detail("- SkipScreenshots: %s", configs.SkipScreenshots)
	log.Detail("- TeamID: %s", configs.TeamID)
	log.Detail("- TeamName: %s", configs.TeamName)
	log.Detail("- Options: %s", configs.Options)

	log.Detail("- UpdateDeliver: %s", configs.UpdateDeliver)
}

func (configs ConfigsModel) validate() error {
	if configs.IpaPath == "" && configs.PkgPath == "" {
		return errors.New("no IpaPath not PkgPath parameter specified")
	}

	if configs.IpaPath != "" {
		if exist, err := pathutil.IsPathExists(configs.IpaPath); err != nil {
			return fmt.Errorf("failed to check if IpaPath exist at: %s, error: %s", configs.IpaPath, err)
		} else if !exist {
			return fmt.Errorf("IpaPath not exist at: %s", configs.IpaPath)
		}
	}

	if configs.PkgPath != "" {
		if exist, err := pathutil.IsPathExists(configs.PkgPath); err != nil {
			return fmt.Errorf("failed to check if PkgPath exist at: %s, error: %s", configs.PkgPath, err)
		} else if !exist {
			return fmt.Errorf("PkgPath not exist at: %s", configs.PkgPath)
		}
	}

	if configs.ItunesconUser == "" {
		return errors.New("no ItunesconUser parameter specified")
	}

	if configs.Password == "" {
		return errors.New("no Password parameter specified")
	}

	if configs.AppID == "" {
		return errors.New("no AppID parameter specified")
	}

	if configs.SubmitForBeta == "" {
		return errors.New("no SubmitForBeta parameter specified")
	}

	if configs.SkipMetadata == "" {
		return errors.New("no SkipMetadata parameter specified")
	}

	if configs.SkipScreenshots == "" {
		return errors.New("no SkipScreenshots parameter specified")
	}

	if configs.UpdateDeliver == "" {
		return errors.New("no UpdateDeliver parameter specified")
	}

	return nil
}

func fail(format string, v ...interface{}) {
	log.Error(format, v...)
	os.Exit(1)
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
	log.Info("Setup")

	startTime := time.Now()

	rubyCmd, err := rubycmd.NewRubyCommandModel()
	if err != nil {
		fail("Failed to create ruby command model, error: %s", err)
	}

	deliverGemName := "deliver"
	installed, err := rubyCmd.IsGemInstalled(deliverGemName, "")
	if err != nil {
		fail("Failed to check if gem (%s) installed, error: %s", err)
	}

	installDeliver := true
	if installed {
		log.Detail("%s already installed", deliverGemName)

		if configs.UpdateDeliver == "no" {
			log.Detail("update %s disabled, setup finished...", deliverGemName)
			installDeliver = false
		} else {
			log.Detail("updating %s...", deliverGemName)
		}
	} else {
		log.Detail("%s NOT yet installed, attempting install...")
	}

	if installDeliver {
		rubyCmd.GemInstall(deliverGemName, "")
	}

	elapsed := time.Since(startTime)

	log.Detail("Setup took %d secounds to complete", elapsed.Seconds)

	//
	// Main
	fmt.Println()
	log.Info("Deploy")

	log.Detail(`**Note:** if your password
contains special characters
and you experience problems, please
consider changing your password
to something with only
alphanumeric characters.`)
	fmt.Println()

	log.Detail(`**Be advised** log.Detail(that this
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
		"--username", configs.ItunesconUser,
		"--app", configs.AppID,
	}
	
	if configs.TeamID != "" {
		args = append(args, "--team_id", configs.TeamID)
	}
	
	if configs.TeamName != "" {
		args = append(args, "--team_name", configs.TeamName)
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

	args = append(args, options...)

	log.Done("$ %s", cmdex.PrintableCommandArgs(false, append([]string{"deliver"}, args...)))

	cmd := cmdex.NewCommand("deliver", args...)

	cmd.SetStdout(os.Stdout)
	cmd.SetStderr(os.Stderr)
	cmd.SetStdin(os.Stdin)
	cmd.AppendEnvs(envs)

	fmt.Println()

	if err := cmd.Run(); err != nil {
		fail("Deploy failed, error: %s", err)
	}

	log.Done("Success")
	log.Detail("The app (.ipa) was successfully uploaded to [iTunes Connect](https://itunesconnect.apple.com), you should see it in the *Prerelease* section on the app's iTunes Connect page!")
	log.Detail("**Don't forget to enable** the **TestFlight Beta Testing** switch on iTunes Connect (on the *Prerelease* tab of the app) if this is a new version of the app!")
}
