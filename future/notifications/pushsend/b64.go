package pushsend

import "encoding/base64"

func decodeB64URL(v string) ([]byte, error) {
	if b, err := base64.RawURLEncoding.DecodeString(v); err == nil {
		return b, nil
	}
	return base64.URLEncoding.DecodeString(v)
}
