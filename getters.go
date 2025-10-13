package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"

	"github.com/deroproject/derohe/rpc"
)

func getFiles(keys map[string]any) (files []file) {
	for k, v := range keys {
		if k == "C" || k == "owner" || k == "." || k == "total" || k == "account" || k == "bucket" {
			// skip C value, owner, init & total value
			continue
		}
		switch t := v.(type) {
		case string:
			b, err := hex.DecodeString(t)
			if err != nil {
				continue
			}

			f := file{Name: k, EOF: binary.BigEndian.Uint64(b)}
			fmt.Printf("%+v\n", f)
			files = append(files, f)
		case float64:
			f := file{Name: k, EOF: uint64(t)}
			fmt.Printf("%+v\n", f)
			files = append(files, f)
		}
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].EOF < files[j].EOF
	})

	return
}
func getDapp(sc rpc.GetSC_Result) (files map[string]struct {
	Content     []byte
	ContentType string
}) {
	// fmt.Println("Smart Contract Code:")
	// fmt.Println(sc.Code)

	var contract index
	fmt.Println("dAPP Index")
	// fmt.Println(sc.VariableStringKeys)

	contract.Files = getFiles(sc.VariableStringKeys)

	chunks := getChunks(sc.VariableUint64Keys)
	// fmt.Println(chunks)
	files = decompressFiles(contract, sc, chunks, make(map[string]struct {
		Content     []byte
		ContentType string
	}))
	return
}
func getData(index map[string]any, scid string) string {

	decoded := decodeKeys(index)

	i, ok := decoded[scid]
	if !ok {
		return ""
	}
	f := i.(float64)
	v := strconv.Itoa(int(f))
	data, ok := decoded[v].(string)
	if !ok || data == "" {
		return ""
	}

	decompressed, err := io.ReadAll(decompressData(decodeData(data)))
	if err != nil {
		panic(err)
	}

	return string(decompressed)
}

func getRandAddr() string {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      "RANDOM ADDRES",
		"method":  "DERO.GetRandomAddress",
	}
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Failed to marshal payload:", err)
		return ""
	}

	resp, err := http.Post(json_rpc, "application/json", bytes.NewReader(jsonBytes))
	if err != nil {
		fmt.Println("Failed to post:", err)
		return ""
	}
	defer resp.Body.Close()

	var result map[string]any
	respBody, _ := io.ReadAll(resp.Body)
	// fmt.Println(string(respBody))
	if err := json.Unmarshal(respBody, &result); err != nil {
		fmt.Println("Failed to unmarshal:", err)
		return ""
	}
	r := result["result"]
	return r.(map[string]any)["address"].([]any)[0].(string)
}
func getIndexOwner(index map[string]any, owners map[uint64]any, scid string) string {
	i, ok := index[scid]
	if !ok {
		return ""
	}
	return owners[uint64(i.(float64))].(string)
}
func getPageOwner(index map[string]any) string {
	i, ok := index["owner"]
	if !ok {
		return ""
	}
	decoded, err := hex.DecodeString(i.(string))
	if err != nil {
		return ""
	}
	return string(decoded)
}
func getSC(scid string) rpc.GetSC_Result {

	resp, err := http.Post(json_rpc, "application/json", bytes.NewBufferString(`
	{
		"jsonrpc": "2.0",
		"id": "GET SC",
		"method": "DERO.GetSC",
		"params": {
			"scid": "`+scid+`",
			"code": true,
			"variables": true
		}
	}
	`))
	if err != nil {
		fmt.Println("Error executing request:", err)
		return rpc.GetSC_Result{}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return rpc.GetSC_Result{}
	}
	type response struct {
		Result rpc.GetSC_Result
	}
	var r response
	if err := json.Unmarshal(body, &r); err != nil {
		fmt.Println("Failed to unmarshal response:", err)
		return rpc.GetSC_Result{}
	}
	return r.Result
}
func getChunks(keys map[uint64]any) (chunks []chunkedCode) {
	fmt.Println("dAPP CODE")
	for line, c := range decodeLines(keys) {
		// fmt.Println(line, c)
		chunks = append(chunks, chunkedCode{
			Line:    line,
			Content: c.(string),
		})
	}
	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].Line < chunks[j].Line
	})

	return
}
