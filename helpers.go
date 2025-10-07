package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"
)

func unescapeLines(b []byte) string {
	var unescaped string
	if err := json.Unmarshal(b, &unescaped); err != nil {
		return err.Error()
	}
	return unescaped
}

func unzipLines(data []byte) []byte {
	reader := bytes.NewReader(data)
	g, err := gzip.NewReader(reader)
	if err != nil {
		panic(err)
	}
	b, err := io.ReadAll(g)
	if err != nil {
		panic(err)
	}
	return b
}

func decodeLines(keys map[uint64]any) map[uint64]any {
	decoded := make(map[uint64]any)
	for line, h := range keys {
		b, err := hex.DecodeString(h.(string))
		if err != nil {
			panic(err)
		}
		decoded[line] = string(b)
	}
	return decoded
}
func decodeData(s string) []byte {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}
func decodeKeys(keys map[string]any) map[string]any {
	decoded := make(map[string]any)
	for line, data := range keys {
		switch d := data.(type) {
		case string:
			b, err := hex.DecodeString(d)
			if err != nil {
				panic(err)
			}
			decoded[line] = string(b)
		case float64:
			decoded[line] = d
		}
	}
	return decoded
}
func decodeChunks(chunks []chunkedCode) []byte {
	var b []byte
	for _, chunk := range chunks {
		// fmt.Println(chunk.Content)
		d, err := base64.StdEncoding.DecodeString(chunk.Content)
		if err != nil {
			panic(err)
		}
		// fmt.Println(string(b))
		b = append(b, d...)
	}
	return b
}

func decompressData(b []byte) *gzip.Reader {
	r := bytes.NewReader(b)
	gz, err := gzip.NewReader(r)
	if err != nil {
		panic(err)
	}
	defer gz.Close()
	return gz
}

func decompressFiles(contract index, chunks []chunkedCode, files map[string]struct {
	Content     []byte
	ContentType string
}) map[string]struct {
	Content     []byte
	ContentType string
} {
	start := uint64(0)
	for i, file := range contract.Files {
		end := file.EOF
		name := file.Name
		// fmt.Println(chunks)
		switch i {
		default:
			b := decodeChunks(chunks[start:end])
			start = end
			// fmt.Println(string(b))
			unzipped := unzipLines(b)
			unescaped := unescapeLines(unzipped)
			if strings.Contains(unescaped, "invalid character") {
				unescaped = string(unzipped)
			}
			// fmt.Println(unescaped)
			mimeType := mime.TypeByExtension(filepath.Ext(file.Name))
			if mimeType == "" {
				mimeType = "application/octet-stream"
			}
			file.Lines = strings.Split(unescaped, "\n")
			files[name] = struct {
				Content     []byte
				ContentType string
			}{
				Content:     []byte(unescaped),
				ContentType: mimeType,
			}
			fmt.Println(string(files[name].ContentType))
		}

	}
	return files
}
