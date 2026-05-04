package util

import (
	"fmt"
	"os"
	"strconv"
)

func MustGetString(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("environment variable %s is required but not set", key)
	}
	return value, nil
}

func MustGetInt(key string) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return 0, fmt.Errorf("environment variable %s is required but not set", key)
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("environment variable %s must be an integer: %v", key, err)
	}
	return intValue, nil
}

func GetIntEnv(key string, defaultValue int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("environment variable %s must be an integer: %v", key, err)
	}
	return intValue, nil
}

func GetStringEnv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func GetBoolEnv(key string, defaultValue bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("environment variable %s must be a boolean: %v", key, err)
	}
	return boolValue, nil
}
