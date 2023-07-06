package ciphertext_test

import (
	"encoding/binary"
	"os"
	"testing"

	"github.com/vela-ssoc/vela-broker/ciphertext"
)

func TestEncrypt(t *testing.T) {
	raw := []byte("ABC")
	ret := ciphertext.Encrypt(raw)
	t.Logf("%s", ret)
}

func TestDecrypt(t *testing.T) {
	enc := []byte("KSUnC1vMCoZkCw==")
	ret, err := ciphertext.Decrypt(enc)
	t.Log(err)
	t.Logf("%s", ret)
}

type Person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestFile(t *testing.T) {
	p := Person{Name: "撒反对十三点三十手动阀实打实", Age: -109}
	enc, err := ciphertext.EncryptJSON(p)
	if err != nil {
		t.Error(err)
		return
	}

	file, err := os.OpenFile("enc.txt", os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.ModePerm)
	if err != nil {
		t.Error(err)
		return
	}
	defer func() { _ = file.Close() }()

	size := len(enc)
	psz := make([]byte, 4)
	binary.BigEndian.PutUint32(psz, uint32(size))

	if _, err = file.Write(enc); err != nil {
		t.Error(err)
		return
	}

	if _, err = file.Write(psz); err != nil {
		t.Error(err)
		return
	}
}

func TestInit(t *testing.T) {
}
