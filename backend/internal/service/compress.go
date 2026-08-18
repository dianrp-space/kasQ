package service

import (
	"bytes"
	"image"
	"image/jpeg"
	"log"
	"strings"

	_ "image/gif"
	_ "image/png"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/webp"
)

const (
	notaCompressAbove = 2 << 20 // 2MB
	notaMaxEdge       = 1920
	notaJPEGQuality   = 80
)

func init() {
	image.RegisterFormat("webp", "RIFF????WEBP", webp.Decode, webp.DecodeConfig)
}

type compressedNota struct {
	data        []byte
	contentType string
	filename    string
}

func prepareNotaUpload(filename, contentType string, data []byte) (compressedNota, error) {
	out := compressedNota{data: data, contentType: contentType, filename: filename}
	if len(data) <= notaCompressAbove {
		return out, nil
	}
	if !isCompressibleImage(filename, contentType, data) {
		return out, nil
	}

	compressed, err := compressNotaImage(data)
	if err != nil {
		log.Printf("nota: skip compress (%s, %d bytes): %v", filename, len(data), err)
		return out, nil
	}
	if len(compressed) >= len(data) {
		return out, nil
	}
	log.Printf("nota: compressed %s %d → %d bytes", filename, len(data), len(compressed))
	out.data = compressed
	out.contentType = "image/jpeg"
	out.filename = replaceExt(filename, ".jpg")
	return out, nil
}

func isCompressibleImage(filename, contentType string, data []byte) bool {
	ct := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	switch ct {
	case "image/jpeg", "image/jpg", "image/png", "image/webp", "image/gif":
		return true
	}
	ext := strings.ToLower(notaObjectExt(filename, contentType))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif":
		return true
	}
	return looksLikeImage(data)
}

func looksLikeImage(data []byte) bool {
	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return true
	}
	if len(data) >= 8 && data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return true
	}
	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return true
	}
	return false
}

func compressNotaImage(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	img = applyJPEGOrientation(data, img)
	img = fitMaxEdge(img, notaMaxEdge)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: notaJPEGQuality}); err != nil {
		return nil, err
	}
	out := buf.Bytes()
	if len(out) <= notaCompressAbove {
		return out, nil
	}

	for q := 70; q >= 55; q -= 5 {
		buf.Reset()
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: q}); err != nil {
			return nil, err
		}
		out = buf.Bytes()
		if len(out) <= notaCompressAbove {
			return out, nil
		}
	}
	return out, nil
}

func fitMaxEdge(src image.Image, maxEdge int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxEdge && h <= maxEdge {
		return src
	}
	scale := float64(maxEdge) / float64(w)
	if h > w {
		scale = float64(maxEdge) / float64(h)
	}
	nw := int(float64(w)*scale + 0.5)
	nh := int(float64(h)*scale + 0.5)
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)
	return dst
}

func replaceExt(filename, ext string) string {
	name := strings.ReplaceAll(filename, "..", "")
	name = strings.ReplaceAll(name, "/", "-")
	if i := strings.LastIndex(name, "."); i > 0 {
		return name[:i] + ext
	}
	if name == "" {
		return "nota" + ext
	}
	return name + ext
}

func applyJPEGOrientation(data []byte, img image.Image) image.Image {
	orient := jpegOrientation(data)
	switch orient {
	case 3:
		return rotate180(img)
	case 6:
		return rotate90(img)
	case 8:
		return rotate270(img)
	default:
		return img
	}
}

func jpegOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}
	i := 2
	for i+4 < len(data) {
		if data[i] != 0xFF {
			break
		}
		marker := data[i+1]
		if marker == 0xDA || marker == 0xD9 {
			break
		}
		if i+3 >= len(data) {
			break
		}
		size := int(data[i+2])<<8 | int(data[i+3])
		if size < 2 || i+2+size > len(data) {
			break
		}
		if marker == 0xE1 {
			seg := data[i+4 : i+2+size]
			if o := parseExifOrientation(seg); o > 0 {
				return o
			}
		}
		i += 2 + size
	}
	return 1
}

func parseExifOrientation(seg []byte) int {
	if len(seg) < 14 || string(seg[:6]) != "Exif\x00\x00" {
		return 0
	}
	tiff := seg[6:]
	var endian binaryEndian
	if string(tiff[:2]) == "MM" {
		endian = bigEndian
	} else if string(tiff[:2]) == "II" {
		endian = littleEndian
	} else {
		return 0
	}
	ifd0 := int(endian.u32(tiff[4:8]))
	if ifd0 < 8 || ifd0+2 > len(tiff) {
		return 0
	}
	n := int(endian.u16(tiff[ifd0 : ifd0+2]))
	for e := 0; e < n; e++ {
		off := ifd0 + 2 + e*12
		if off+12 > len(tiff) {
			break
		}
		tag := endian.u16(tiff[off : off+2])
		if tag != 0x0112 {
			continue
		}
		return int(endian.u16(tiff[off+8 : off+10]))
	}
	return 0
}

type binaryEndian int

const (
	bigEndian binaryEndian = iota
	littleEndian
)

func (e binaryEndian) u16(b []byte) uint16 {
	if e == bigEndian {
		return uint16(b[0])<<8 | uint16(b[1])
	}
	return uint16(b[1])<<8 | uint16(b[0])
}

func (e binaryEndian) u32(b []byte) uint32 {
	if e == bigEndian {
		return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
	}
	return uint32(b[3])<<24 | uint32(b[2])<<16 | uint32(b[1])<<8 | uint32(b[0])
}

func rotate90(src image.Image) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, h, w))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.Set(h-1-y, x, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func rotate180(src image.Image) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.Set(w-1-x, h-1-y, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func rotate270(src image.Image) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, h, w))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.Set(y, w-1-x, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
