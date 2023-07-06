package ciphertext

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"hash/adler32"
	"io"
	"os"
)

var ErrDecrypt = errors.New("decrypt failed")

// Decrypt 解密
func Decrypt(enc []byte) ([]byte, error) {
	// base64解码
	dsz := base64.StdEncoding.DecodedLen(len(enc))
	raw := make([]byte, dsz)
	n, err := base64.StdEncoding.Decode(raw, enc)
	if err != nil || n < 7 {
		return nil, ErrDecrypt
	}
	mask := raw[n-1]
	for i, b := range raw {
		raw[i] = mask ^ b
	}

	rsz := n - 7
	load := raw[rsz : n-1]
	// vc[2] mask2 vc[3] vc[0] vc[1] mask1
	vc := make([]byte, 4)
	var mask1, mask2 byte
	vc[0], vc[1], vc[2], vc[3], mask1, mask2 = load[3], load[4], load[0], load[2], load[5], load[1]
	for i := 0; i < rsz; i += 2 {
		if i+1 >= rsz {
			raw[i] ^= mask1
			break
		}
		// 奇数位mask1 偶数位mask2
		raw[i], raw[i+1] = raw[i+1]^mask1, raw[i]^mask2
	}
	raw = raw[:rsz]
	sum := binary.BigEndian.Uint32(vc)
	if adler32.Checksum(raw) != sum {
		return nil, ErrDecrypt
	}

	return raw, nil
}

// DecryptJSON 解密后反序列化为struct
func DecryptJSON(enc []byte, v any) error {
	raw, err := Decrypt(enc)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, v)
}

// DecryptFile 从文件的payload读取加密数据并反序列化到struct
func DecryptFile(path string, v any) error {
	self, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = self.Close() }()

	// 读取文件的总大小
	var size int64
	if stat, err := self.Stat(); err == nil {
		size = stat.Size()
	}
	// 4位校验码 3个mask 4位payload长度
	if size < 11 {
		return ErrDecrypt
	}

	// 只读取长度
	if _, err = self.Seek(-4, io.SeekEnd); err != nil {
		return err
	}
	data := make([]byte, 4)
	if _, err = self.Read(data); err != nil {
		return err
	}
	// 扰乱顺序 3012 回归正常
	data[0], data[1], data[2], data[3] = data[1], data[2], data[3], data[0]

	psz := binary.BigEndian.Uint32(data)

	if size-4 < int64(psz) {
		return ErrDecrypt
	}
	if _, err = self.Seek(-int64(psz+4), io.SeekEnd); err != nil {
		return err
	}
	enc := make([]byte, psz)
	if _, err = self.Read(enc); err != nil {
		return err
	}

	return DecryptJSON(enc, v)
}
