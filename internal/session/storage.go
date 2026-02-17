package session

import (
	"os"
	"path/filepath"
	"strings"
)

const storeRoot = "store"
const sessionPrefix = "session_"
const tokenFileName = "token.txt"

func EnsureStoreRoot() error {
	return os.MkdirAll(storeRoot, 0o755)
}

func SessionDir(id string) string {
	return filepath.Join(storeRoot, sessionPrefix+id)
}

func TokenPath(id string) string {
	return filepath.Join(SessionDir(id), tokenFileName)
}

func ListSessionIDs() ([]string, error) {
	if err := EnsureStoreRoot(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(storeRoot)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, sessionPrefix) {
			ids = append(ids, strings.TrimPrefix(name, sessionPrefix))
		}
	}
	return ids, nil
}

func ReadToken(id string) (string, error) {
	data, err := os.ReadFile(TokenPath(id))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func WriteToken(id string, token string) error {
	return os.WriteFile(TokenPath(id), []byte(token+"\n"), 0o600)
}
