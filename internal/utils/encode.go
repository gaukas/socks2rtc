package utils

import (
	b64 "encoding/base64"
)

func ToBase64(in []byte) []byte {
	out := make([]byte, b64.StdEncoding.EncodedLen(len(in)))
	b64.StdEncoding.Encode(out, in)
	return out
}

func FromBase64(in []byte) ([]byte, error) {
	out := make([]byte, b64.StdEncoding.DecodedLen(len(in)))
	n, err := b64.StdEncoding.Decode(out, in)
	if err != nil {
		return nil, err
	}
	return out[:n], nil
}
