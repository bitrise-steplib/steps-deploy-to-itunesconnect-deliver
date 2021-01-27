package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
)

func fetchPrivateKey(apiKeyPath string) ([]byte, string, error) {
	// see these in the altool's man page
	var keyPaths = []string{
		filepath.Join(os.Getenv("HOME"), ".appstoreconnect/private_keys"),
		filepath.Join(os.Getenv("HOME"), ".private_keys"),
		filepath.Join(os.Getenv("HOME"), "private_keys"),
		"./private_keys",
	}

	fileURL, err := url.Parse(apiKeyPath)
	if err != nil {
		return nil, "", err
	}

	keyID := getKeyID(fileURL)

	keyPath, keyExists, err := getKeyPath(keyID, keyPaths)
	if err != nil {
		return nil, "", err
	}

	if !keyExists {
		if err := copyOrDownloadFile(fileURL, keyPath); err != nil {
			return nil, "", err
		}
	}

	key, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, "", err
	}

	return key, keyID, nil
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
