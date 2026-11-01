package client

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
)

var srunBase64 = base64.NewEncoding("LVoJPiCN2R8G90yg+hmFHuacZ1OWMnrsSTXkYpUq/3dlbfKwv6xztjI7DeBE45QA")

func passwordHash(token string) string {
	// Preserve the existing Rust client's SRUN dialect: HMAC-MD5 of the
	// empty message. The account password is carried in encrypted info.
	return hex.EncodeToString(hmac.New(md5.New, []byte(token)).Sum(nil))
}

func loginInfo(username, password, ip, acid, token string) string {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(map[string]string{
		"username": username,
		"password": password,
		"ip":       ip,
		"acid":     acid,
		"enc_ver":  "srun_bx1",
	})

	payload := bytes.TrimSuffix(buffer.Bytes(), []byte{'\n'})
	return "{SRBX1}" + srunBase64.EncodeToString(xencode(payload, []byte(token)))
}

func checksum(token, username, hash, acid, ip, info string) string {
	value := sha1.Sum([]byte(token + username + token + hash + token + acid + token + ip + token + "200" + token + "1" + token + info))
	return hex.EncodeToString(value[:])
}

func logoutSign(timestamp, username, ip string) string {
	value := sha1.Sum([]byte(timestamp + username + ip + "1" + timestamp))
	return hex.EncodeToString(value[:])
}

func xencode(message, key []byte) []byte {
	words := make([]uint32, (len(message)+3)/4+1)
	for i, b := range message {
		words[i/4] |= uint32(b) << (8 * (i % 4))
	}
	words[len(words)-1] = uint32(len(message))

	var keys [4]uint32
	for i, b := range key[:min(len(key), 16)] {
		keys[i/4] |= uint32(b) << (8 * (i % 4))
	}

	z, sum := words[len(words)-1], uint32(0)
	for round := 0; round < 6+52/len(words); round++ {
		sum += 0x9e3779b9
		e := (sum >> 2) & 3
		for p := range words {
			y := words[(p+1)%len(words)]

			// This is SRUN's expression as used by biternet-client, not the
			// textbook XXTEA MX expression. uint32 arithmetic wraps naturally.
			z = ((z >> 5) ^ (y << 2)) +
				(((y >> 3) ^ (z << 4)) ^ (sum ^ y)) +
				(keys[(uint32(p)&3)^e] ^ z) + words[p]
			words[p] = z
		}
	}

	result := make([]byte, 4*len(words))
	for i, word := range words {
		binary.LittleEndian.PutUint32(result[4*i:], word)
	}

	return result
}
