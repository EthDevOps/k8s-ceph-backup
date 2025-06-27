package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type GPGClient struct {
	gpgPath    string
	recipient  string
	keyring    string
	trustModel string
}

func NewGPGClient() *GPGClient {
	return &GPGClient{
		gpgPath:    viper.GetString("gpg.path"),
		recipient:  viper.GetString("gpg.recipient"),
		keyring:    viper.GetString("gpg.keyring"),
		trustModel: viper.GetString("gpg.trust_model"),
	}
}

func (g *GPGClient) EncryptFile(inputPath string) (string, error) {
	log.Debugf("Encrypting file: %s", inputPath)

	if g.gpgPath == "" {
		g.gpgPath = "gpg"
	}

	if g.recipient == "" {
		return "", fmt.Errorf("GPG recipient not configured")
	}

	outputPath := inputPath + ".gpg"

	args := []string{
		"--encrypt",
		"--armor",
		"--recipient", g.recipient,
		"--output", outputPath,
	}

	if g.keyring != "" {
		args = append(args, "--keyring", g.keyring)
	}

	if g.trustModel != "" {
		args = append(args, "--trust-model", g.trustModel)
	} else {
		args = append(args, "--trust-model", "always")
	}

	args = append(args, inputPath)

	log.Debugf("Running GPG command: %s %v", g.gpgPath, args)

	cmd := exec.Command(g.gpgPath, args...)
	
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("GPG encryption failed: %w", err)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat encrypted file: %w", err)
	}

	log.Infof("Successfully encrypted file to %s (size: %d bytes)", outputPath, info.Size())
	return outputPath, nil
}

func (g *GPGClient) DecryptFile(inputPath string) (string, error) {
	log.Debugf("Decrypting file: %s", inputPath)

	if g.gpgPath == "" {
		g.gpgPath = "gpg"
	}

	outputPath := filepath.Dir(inputPath) + "/" + filepath.Base(inputPath)
	if filepath.Ext(outputPath) == ".gpg" {
		outputPath = outputPath[:len(outputPath)-4]
	}

	args := []string{
		"--decrypt",
		"--output", outputPath,
	}

	if g.keyring != "" {
		args = append(args, "--keyring", g.keyring)
	}

	args = append(args, inputPath)

	log.Debugf("Running GPG decrypt command: %s %v", g.gpgPath, args)

	cmd := exec.Command(g.gpgPath, args...)
	
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("GPG decryption failed: %w", err)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat decrypted file: %w", err)
	}

	log.Infof("Successfully decrypted file to %s (size: %d bytes)", outputPath, info.Size())
	return outputPath, nil
}

func (g *GPGClient) ListKeys() error {
	log.Debug("Listing GPG keys")

	if g.gpgPath == "" {
		g.gpgPath = "gpg"
	}

	args := []string{"--list-keys"}

	if g.keyring != "" {
		args = append(args, "--keyring", g.keyring)
	}

	cmd := exec.Command(g.gpgPath, args...)
	
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to list GPG keys: %w", err)
	}

	return nil
}

func (g *GPGClient) ValidateRecipient() error {
	log.Debugf("Validating GPG recipient: %s", g.recipient)

	if g.recipient == "" {
		return fmt.Errorf("GPG recipient not configured")
	}

	if g.gpgPath == "" {
		g.gpgPath = "gpg"
	}

	args := []string{"--list-keys", g.recipient}

	if g.keyring != "" {
		args = append(args, "--keyring", g.keyring)
	}

	cmd := exec.Command(g.gpgPath, args...)
	
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("GPG recipient %s not found or invalid: %w", g.recipient, err)
	}

	log.Debugf("GPG recipient validation successful: %s", string(output))
	return nil
}