package input

import (
	"fmt"

	"github.com/bitrise-io/go-utils/pathutil"
)

// ValidateWithOptions ...
func ValidateWithOptions(value string, options ...string) error {
	for _, option := range options {
		if option == value {
			return nil
		}
	}
	return fmt.Errorf("invalid parameter: %s, available: %v", value, options)
}

// ValidateIfNotEmpty ...
func ValidateIfNotEmpty(input string) error {
	if input == "" {
		return fmt.Errorf("parameter not specified")
	}
	return nil
}

// ValidateIfPathExists ...
func ValidateIfPathExists(input string) error {
	if err := ValidateIfNotEmpty(input); err != nil {
		return err
	}
	if exist, err := pathutil.IsPathExists(input); err != nil {
		return fmt.Errorf("failed to check if path exist at: %s, error: %s", input, err)
	} else if !exist {
		return fmt.Errorf("path not exist at: %s", input)
	}
	return nil
}

// SecureInput ...
func SecureInput(input string) string {
	if input != "" {
		return "***"
	}
	return ""
}
