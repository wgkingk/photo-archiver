package verifier

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

const (
	ModeSize = "size"
	ModeHash = "hash"
)

func Verify(srcPath, dstPath, mode string, srcHash string) error {
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}
	dstInfo, err := os.Stat(dstPath)
	if err != nil {
		return fmt.Errorf("stat target: %w", err)
	}
	if srcInfo.Size() != dstInfo.Size() {
		return fmt.Errorf("size mismatch: src=%d dst=%d", srcInfo.Size(), dstInfo.Size())
	}
	if mode != ModeHash {
		return nil
	}
	if srcHash == "" {
		srcHash, err = HashFile(srcPath)
		if err != nil {
			return fmt.Errorf("hash source: %w", err)
		}
	}
	dstHash, err := HashFile(dstPath)
	if err != nil {
		return fmt.Errorf("hash target: %w", err)
	}
	if srcHash != dstHash {
		return fmt.Errorf("hash mismatch: src=%s dst=%s", srcHash, dstHash)
	}
	return nil
}

func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
