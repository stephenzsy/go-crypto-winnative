// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

//go:build windows
// +build windows

package cng

import (
	"bytes"
	"crypto/cipher"
	"testing"
)

var key = []byte("D249BF6DEC97B1EBD69BC4D6B3A3C49D")

func TestNewGCMNonce(t *testing.T) {
	ci, err := NewAESCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	c := ci.(*aesCipher)
	_, err = c.NewGCM(gcmStandardNonceSize-1, gcmTagSize-1)
	if err == nil {
		t.Error("expected error for non-standard tag and nonce size at the same time, got none")
	}
	_, err = c.NewGCM(gcmStandardNonceSize-1, gcmTagSize)
	if err != nil {
		t.Errorf("expected no error for non-standard nonce size with standard tag size, got: %#v", err)
	}
	_, err = c.NewGCM(gcmStandardNonceSize, gcmTagSize-1)
	if err != nil {
		t.Errorf("expected no error for standard tag size, got: %#v", err)
	}
	_, err = c.NewGCM(gcmStandardNonceSize, gcmTagSize)
	if err != nil {
		t.Errorf("expected no error for standard tag / nonce size, got: %#v", err)
	}
}

func TestSealAndOpen(t *testing.T) {
	ci, err := NewAESCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	c := ci.(*aesCipher)
	gcm, err := c.NewGCM(gcmStandardNonceSize, gcmTagSize)
	if err != nil {
		t.Fatal(err)
	}
	nonce := []byte{0x91, 0xc7, 0xa7, 0x54, 0x52, 0xef, 0x10, 0xdb, 0x91, 0xa8, 0x6c, 0xf9}
	plainText := []byte{0x01, 0x02, 0x03}
	additionalData := []byte{0x05, 0x05, 0x07}
	sealed := gcm.Seal(nil, nonce, plainText, additionalData)
	decrypted, err := gcm.Open(nil, nonce, sealed, additionalData)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(decrypted, plainText) {
		t.Errorf("unexpected decrypted result\ngot: %#v\nexp: %#v", decrypted, plainText)
	}
	// Test with no additional data.
	sealed = gcm.Seal(nil, nonce, plainText, []byte{})
	decrypted, err = gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(decrypted, plainText) {
		t.Errorf("unexpected decrypted result\ngot: %#v\nexp: %#v", decrypted, plainText)
	}
}

func TestSealAndOpenTLS(t *testing.T) {
	ci, err := NewAESCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := NewGCMTLS(ci)
	if err != nil {
		t.Fatal(err)
	}
	nonce := [12]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	nonce1 := [12]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	nonce9 := [12]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 9}
	nonce10 := [12]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 10}
	nonceMax := [12]byte{0, 0, 0, 0, 255, 255, 255, 255, 255, 255, 255, 255}
	plainText := []byte{0x01, 0x02, 0x03}
	additionalData := make([]byte, 13)
	additionalData[11] = byte(len(plainText) >> 8)
	additionalData[12] = byte(len(plainText))
	sealed := gcm.Seal(nil, nonce[:], plainText, additionalData)
	assertPanic(t, func() {
		gcm.Seal(nil, nonce[:], plainText, additionalData)
	})
	sealed1 := gcm.Seal(nil, nonce1[:], plainText, additionalData)
	gcm.Seal(nil, nonce10[:], plainText, additionalData)
	assertPanic(t, func() {
		gcm.Seal(nil, nonce9[:], plainText, additionalData)
	})
	assertPanic(t, func() {
		gcm.Seal(nil, nonceMax[:], plainText, additionalData)
	})
	if bytes.Equal(sealed, sealed1) {
		t.Errorf("different nonces should produce different outputs\ngot: %#v\nexp: %#v", sealed, sealed1)
	}
	decrypted, err := gcm.Open(nil, nonce[:], sealed, additionalData)
	if err != nil {
		t.Error(err)
	}
	decrypted1, err := gcm.Open(nil, nonce1[:], sealed1, additionalData)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(decrypted, plainText) {
		t.Errorf("unexpected decrypted result\ngot: %#v\nexp: %#v", decrypted, plainText)
	}
	if !bytes.Equal(decrypted, decrypted1) {
		t.Errorf("unexpected decrypted result\ngot: %#v\nexp: %#v", decrypted, decrypted1)
	}
}

func TestSealAndOpenAuthenticationError(t *testing.T) {
	ci, err := NewAESCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	c := ci.(*aesCipher)
	gcm, err := c.NewGCM(gcmStandardNonceSize, gcmTagSize)
	if err != nil {
		t.Fatal(err)
	}
	nonce := []byte{0x91, 0xc7, 0xa7, 0x54, 0x52, 0xef, 0x10, 0xdb, 0x91, 0xa8, 0x6c, 0xf9}
	plainText := []byte{0x01, 0x02, 0x03}
	additionalData := []byte{0x05, 0x05, 0x07}
	sealed := gcm.Seal(nil, nonce, plainText, additionalData)
	_, err = gcm.Open(nil, nonce, sealed, nil)
	if err != errOpen {
		t.Errorf("expected authentication error, got: %#v", err)
	}
}

func assertPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	f()
}

func TestSealPanic(t *testing.T) {
	ci, err := NewAESCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	c := ci.(*aesCipher)
	gcm, err := c.NewGCM(gcmStandardNonceSize, gcmTagSize)
	if err != nil {
		t.Fatal(err)
	}
	assertPanic(t, func() {
		gcm.Seal(nil, make([]byte, gcmStandardNonceSize-1), []byte{0x01, 0x02, 0x03}, nil)
	})
	assertPanic(t, func() {
		// maxInt is implemented as math.MaxInt, but this constant
		// is only available since go1.17.
		// TODO: use math.MaxInt once go1.16 is no longer supported.
		maxInt := int((^uint(0)) >> 1)
		gcm.Seal(nil, make([]byte, gcmStandardNonceSize), make([]byte, maxInt), nil)
	})
}

func TestAESInvalidKeySize(t *testing.T) {
	_, err := NewAESCipher([]byte{1})
	if err == nil {
		t.Error("error expected")
	}
}

func TestEncryptAndDecrypt(t *testing.T) {
	ci, err := NewAESCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	plainText := make([]byte, ci.BlockSize())
	plainText[1] = 1
	plainText[15] = 15
	cipherText := make([]byte, ci.BlockSize())
	decrypted := make([]byte, ci.BlockSize())
	ci.Encrypt(cipherText, plainText)
	ci.Decrypt(decrypted, cipherText)
	if !bytes.Equal(decrypted, plainText) {
		t.Errorf("unexpected decrypted result\ngot: %#v\nexp: %#v", decrypted, plainText)
	}
}

func TestCBCBlobEncryptBasicBlockEncryption(t *testing.T) {
	key := []byte{0x24, 0xcd, 0x8b, 0x13, 0x37, 0xc5, 0xc1, 0xb1, 0x0, 0xbb, 0x27, 0x40, 0x4f, 0xab, 0x5f, 0x7b, 0x2d, 0x0, 0x20, 0xf5, 0x1, 0x84, 0x4, 0xbf, 0xe3, 0xbd, 0xa1, 0xc4, 0xbf, 0x61, 0x2f, 0xc5}
	iv := []byte{0x91, 0xc7, 0xa7, 0x54, 0x52, 0xef, 0x10, 0xdb, 0x91, 0xa8, 0x6c, 0xf9, 0x79, 0xd5, 0xac, 0x74}

	block, err := NewAESCipher(key)
	if err != nil {
		t.Errorf("expected no error for aes.NewCipher, got: %s", err)
	}

	blockSize := block.BlockSize()
	if blockSize != 16 {
		t.Errorf("unexpected block size, expected 16 got: %d", blockSize)
	}
	var encryptor cipher.BlockMode
	if c, ok := block.(*aesCipher); ok {
		encryptor = c.NewCBCEncrypter(iv)
		if encryptor == nil {
			t.Error("unable to create new CBC encrypter")
		}
	}

	encrypted := make([]byte, 32)

	// First block. 16 bytes.
	srcBlock1 := bytes.Repeat([]byte{0x01}, 16)
	encryptor.CryptBlocks(encrypted, srcBlock1)
	if !bytes.Equal([]byte{
		0x14, 0xb7, 0x3e, 0x2f, 0xd9, 0xe7, 0x69, 0x7e, 0xb7, 0xd2, 0xc3, 0x5b, 0x31, 0x9c, 0xf0, 0x59,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}, encrypted) {
		t.Error("unexpected CryptBlocks result for first block")
	}

	// Second block. 16 bytes.
	srcBlock2 := bytes.Repeat([]byte{0x02}, 16)
	encryptor.CryptBlocks(encrypted[16:], srcBlock2)
	if !bytes.Equal([]byte{
		0x14, 0xb7, 0x3e, 0x2f, 0xd9, 0xe7, 0x69, 0x7e, 0xb7, 0xd2, 0xc3, 0x5b, 0x31, 0x9c, 0xf0, 0x59,
		0xbb, 0xd4, 0x95, 0x25, 0x21, 0x56, 0x87, 0x3b, 0xe6, 0x22, 0xe8, 0xd0, 0x19, 0xa8, 0xed, 0xcd,
	}, encrypted) {
		t.Error("unexpected CryptBlocks result for second block")
	}

	var decrypter cipher.BlockMode
	if c, ok := block.(*aesCipher); ok {
		decrypter = c.NewCBCDecrypter(iv)
		if decrypter == nil {
			t.Error("unable to create new CBC decrypter")
		}
	}
	plainText := append(srcBlock1, srcBlock2...)
	decrypted := make([]byte, len(plainText))
	decrypter.CryptBlocks(decrypted, encrypted[:16])
	decrypter.CryptBlocks(decrypted[16:], encrypted[16:])
	if !bytes.Equal(decrypted, plainText) {
		t.Errorf("unexpected decrypted result\ngot: %#v\nexp: %#v", decrypted, plainText)
	}
}

func TestCBCDecryptSimple(t *testing.T) {
	key := []byte{
		0x24, 0xcd, 0x8b, 0x13, 0x37, 0xc5, 0xc1, 0xb1,
		0x0, 0xbb, 0x27, 0x40, 0x4f, 0xab, 0x5f, 0x7b,
		0x2d, 0x0, 0x20, 0xf5, 0x1, 0x84, 0x4, 0xbf,
		0xe3, 0xbd, 0xa1, 0xc4, 0xbf, 0x61, 0x2f, 0xc5,
	}

	block, err := NewAESCipher(key)
	if err != nil {
		t.Fatal(err)
	}

	iv := []byte{
		0x91, 0xc7, 0xa7, 0x54, 0x52, 0xef, 0x10, 0xdb,
		0x91, 0xa8, 0x6c, 0xf9, 0x79, 0xd5, 0xac, 0x74,
	}
	var encrypter, decrypter cipher.BlockMode
	if c, ok := block.(*aesCipher); ok {
		encrypter = c.NewCBCEncrypter(iv)
		if encrypter == nil {
			t.Error("unable to create new CBC encrypter")
		}
		decrypter = c.NewCBCDecrypter(iv)
		if decrypter == nil {
			t.Error("unable to create new CBC decrypter")
		}
	}

	plainText := []byte{
		0x54, 0x68, 0x65, 0x72, 0x65, 0x20, 0x69, 0x73,
		0x20, 0x6f, 0x6e, 0x6c, 0x79, 0x20, 0x6f, 0x6e,
		0x65, 0x20, 0x4c, 0x6f, 0x72, 0x64, 0x20, 0x6f,
		0x66, 0x20, 0x74, 0x68, 0x65, 0x20, 0x52, 0x69,
		0x6e, 0x67, 0x2c, 0x20, 0x6f, 0x6e, 0x6c, 0x79,
		0x20, 0x6f, 0x6e, 0x65, 0x20, 0x77, 0x68, 0x6f,
		0x20, 0x63, 0x61, 0x6e, 0x20, 0x62, 0x65, 0x6e,
		0x64, 0x20, 0x69, 0x74, 0x20, 0x74, 0x6f, 0x20,
		0x68, 0x69, 0x73, 0x20, 0x77, 0x69, 0x6c, 0x6c,
		0x2e, 0x20, 0x41, 0x6e, 0x64, 0x20, 0x68, 0x65,
		0x20, 0x64, 0x6f, 0x65, 0x73, 0x20, 0x6e, 0x6f,
		0x74, 0x20, 0x73, 0x68, 0x61, 0x72, 0x65, 0x20,
		0x70, 0x6f, 0x77, 0x65, 0x72, 0x2e, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	cipherText := make([]byte, len(plainText))

	encrypter.CryptBlocks(cipherText, plainText[:64])
	encrypter.CryptBlocks(cipherText[64:], plainText[64:])

	expectedCipherText := []byte{
		23, 51, 192, 210, 170, 124, 30, 218,
		176, 54, 70, 132, 141, 124, 3, 152,
		47, 3, 37, 81, 187, 101, 197, 94,
		11, 38, 128, 60, 112, 20, 235, 130,
		111, 236, 176, 99, 121, 6, 221, 181,
		190, 228, 150, 177, 218, 3, 196, 0,
		5, 141, 169, 151, 3, 161, 64, 244,
		231, 237, 252, 143, 111, 37, 68, 70,
		11, 137, 220, 243, 195, 90, 182, 83,
		96, 80, 122, 14, 93, 178, 62, 159,
		25, 162, 200, 155, 21, 150, 6, 101,
		21, 234, 12, 74, 190, 213, 159, 220,
		111, 184, 94, 169, 188, 93, 38, 150,
		3, 208, 185, 201, 212, 246, 238, 181,
	}

	if !bytes.Equal(expectedCipherText, cipherText) {
		t.Fail()
	}

	decrypted := make([]byte, len(plainText))

	decrypter.CryptBlocks(decrypted, cipherText[:64])
	decrypter.CryptBlocks(decrypted[64:], cipherText[64:])

	if len(decrypted) != len(plainText) {
		t.Fail()
	}

	if !bytes.Equal(plainText, decrypted) {
		t.Errorf("decryption incorrect\nexp %v, got %v\n", plainText, decrypted)
	}
}
