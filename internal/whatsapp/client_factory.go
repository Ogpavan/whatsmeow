package whatsapp

import (
	"context"
	"os"
	"path/filepath"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
	_ "github.com/mattn/go-sqlite3"
)

func NewClient(ctx context.Context, sessionDir string, handler func(interface{})) (*whatsmeow.Client, error) {
	if err := ensureDir(sessionDir); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(sessionDir, "whatsapp.db")
	container, err := sqlstore.New(ctx, "sqlite3", "file:"+dbPath+"?_foreign_keys=on", waLog.Stdout("wa-mvp-api/db", "INFO", true))
	if err != nil {
		return nil, err
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, err
	}
	if deviceStore == nil {
		deviceStore = container.NewDevice()
	}

	client := whatsmeow.NewClient(deviceStore, nil)
	if handler != nil {
		client.AddEventHandler(handler)
	}

	return client, nil
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}
