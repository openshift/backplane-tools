package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
)

const FedoraSigningKeyURL string = "https://fedoraproject.org/fedora.gpg"

func VerifyGPGSignature(targetFilePath, signatureFilePath string) error {
	targetFile, err := os.Open(targetFilePath)
	if err != nil {
		return fmt.Errorf("failed to open file '%s': %w", targetFilePath, err)
	}
	defer func() {
		closeErr := targetFile.Close()
		if closeErr != nil {
			fmt.Fprintf(os.Stderr, "failed to close file '%s': %v\n", targetFilePath, err)
		}
	}()

	signatureFile, err := os.Open(signatureFilePath)
	if err != nil {
		return fmt.Errorf("failed to open file '%s': %w", signatureFilePath, err)
	}
	defer func() {
		closeErr := signatureFile.Close()
		if closeErr != nil {
			fmt.Fprintf(os.Stderr, "failed to close '%s': %v\n", signatureFilePath, err)
		}
	}()

	fedoraKey, err := GetFedoraGPGKeys()
	if err != nil {
		return fmt.Errorf("failed to retrieve Fedora GPG signing keys: %w", err)
	}
	defer func() {
		closeErr := fedoraKey.Close()
		if closeErr != nil {
			fmt.Fprintf(os.Stderr, "failed to close fedora signing key: %v\n", err)
		}
	}()

	keyRing, err := openpgp.ReadKeyRing(fedoraKey)
	if err != nil {
		return fmt.Errorf("failed to read Fedora GPG signing keys: %w", err)
	}

	_, err = openpgp.CheckArmoredDetachedSignature(keyRing, targetFile, signatureFile, &packet.Config{})
	if err != nil {
		return fmt.Errorf("failed to verify file signature: %w", err)
	}

	return nil
}

func GetFedoraGPGKeys() (io.ReadCloser, error) {
	resp, err := http.Get(FedoraSigningKeyURL)
	if err != nil {
		return nil, fmt.Errorf("failed to GET '%s': %w", FedoraSigningKeyURL, err)
	}

	return resp.Body, nil
}
