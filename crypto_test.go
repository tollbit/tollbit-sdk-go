package tollbit

import "testing"

func TestEncryption(t *testing.T) {
	helloString := "hello world"
	secretKey := "64280b7c9897a66cd1062596518fdf2992361f6477300562dc41d6d108359de2"
	encrypt, err := Encrypt([]byte(helloString), secretKey)
	if err != nil {
		t.Fatal()
	}
	decrypt, err := Decrypt(encrypt, secretKey)
	if err != nil {
		t.Fatal()
	}
	if string(decrypt) != helloString {
		t.Fatal()
	}
}
