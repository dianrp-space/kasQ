package service

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"math/rand"
	"testing"
)

func TestPrepareNotaUploadSkipsSmallFiles(t *testing.T) {
	data := bytes.Repeat([]byte{0xFF, 0xD8, 0xFF}, 100)
	got, err := prepareNotaUpload("nota.jpg", "image/jpeg", data)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.data) != len(data) {
		t.Fatalf("small file should stay as-is, got %d", len(got.data))
	}
}

func TestCompressNotaImageShrinksLargeJPEG(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2200, 1800))
	rng := rand.New(rand.NewSource(42))
	for y := 0; y < 1800; y++ {
		for x := 0; x < 2200; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(rng.Intn(256)),
				G: uint8(rng.Intn(256)),
				B: uint8(rng.Intn(256)),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	raw := buf.Bytes()
	if len(raw) <= notaCompressAbove {
		t.Fatalf("fixture too small: %d", len(raw))
	}

	got, err := prepareNotaUpload("nota.jpg", "image/jpeg", raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.contentType != "image/jpeg" {
		t.Fatalf("content type: %s", got.contentType)
	}
	if len(got.data) >= len(raw) {
		t.Fatalf("expected smaller jpeg, %d >= %d", len(got.data), len(raw))
	}
}
