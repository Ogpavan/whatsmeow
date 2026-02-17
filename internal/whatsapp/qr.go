package whatsapp

import (
	"encoding/base64"

	qrcode "github.com/skip2/go-qrcode"
)

func QRToBase64PNG(qr string) (string, error) {
	png, err := qrcode.Encode(qr, qrcode.Medium, 256)
	if err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(png)
	return "data:image/png;base64," + encoded, nil
}
