package utils

import (
	"fmt"
	"os"
)

func GetAccessKeyOrSecretFromEnvOrPath(envVar string, filePath string) (string, error) {
	if envVar != "" {
		return os.Getenv(envVar), nil
	}
	if filePath != "" {
		bytes, err := os.ReadFile(filePath)
		if err != nil {
			return "", err
		}
		return string(bytes), nil
	}
	return "", fmt.Errorf("no env var or file path provided")
}
