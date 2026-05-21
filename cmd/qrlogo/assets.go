package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/png"
)

// lunarQRLogo is the brand-preset logo bundled into the binary so
// `qrlogo -default` works regardless of the caller's working
// directory. The bytes are the PNG asset under assets/.
//
//go:embed assets/lunar-qr-logo.png
var lunarQRLogo []byte

// decodeLunarQRLogo returns the bundled Lunar QR logo as an
// image.Image. The embedded PNG is decoded fresh on every call so
// callers can mutate the returned image without poisoning future
// readers; the byte slice itself is shared and read-only.
func decodeLunarQRLogo() (image.Image, error) {
	img, err := png.Decode(bytes.NewReader(lunarQRLogo))
	if err != nil {
		return nil, fmt.Errorf("decoding embedded lunar-qr-logo.png: %w", err)
	}
	return img, nil
}
