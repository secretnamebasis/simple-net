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

	"github.com/deroproject/derohe/rpc"
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

var binaryExts = map[string]struct{}{
	".pdf":  {},
	".png":  {},
	".jpg":  {},
	".jpeg": {},
	".gif":  {},
	".zip":  {},
	".exe":  {},
	".tar":  {},
	".gz":   {},
	".mp3":  {},
	".mp4":  {},
	".mov":  {},
	".avi":  {},
	".webp": {},
}

func isBinaryFile(ext string) bool {
	_, ok := binaryExts[strings.ToLower(ext)]
	return ok
}
func decompressFiles(contract index, sc rpc.GetSC_Result, chunks []chunkedCode, files map[string]struct {
	Content     []byte
	ContentType string
}) map[string]struct {
	Content     []byte
	ContentType string
} {

	start := uint64(0)

	for i, f := range contract.Files {
		end := f.EOF
		name := f.Name
		var b []byte

		for _, each := range chunks {

			if each.Line < start || each.Line >= end || each.Content == "" {
				continue
			} else {
				fmt.Println(f.Name, each)
				d, err := base64.StdEncoding.DecodeString(each.Content)
				if err != nil {
					continue
				}
				// fmt.Println(string(b))
				b = append(b, d...)
			}
		}

		if b == nil {
			continue
		}

		start = end
		// fmt.Println(string(b))
		unzipped := unzipLines(b)
		ext := filepath.Ext(name)
		var content []byte
		if isBinaryFile(ext) {
			content = unzipped
		} else {
			e := unescapeLines(unzipped)
			content = []byte(e)
		}

		fmt.Println(string(content))
		contract.Files[i].Lines = strings.Split(string(content), "\n")

		mimeType := mime.TypeByExtension(filepath.Ext(name))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		files[name] = struct {
			Content     []byte
			ContentType string
		}{
			Content:     content,
			ContentType: mimeType,
		}
		// fmt.Println(string(files[name].ContentType))
		if len(f.Lines) == 0 {
			contract.Files[i] = file{}
		}
	}

	return files
}
