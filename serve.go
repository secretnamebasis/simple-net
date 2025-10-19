package main

import (
	"fmt"
	"mime"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/civilware/epoch"
	"github.com/deroproject/derohe/rpc"
)

var (
	port  = "8080"
	files map[string]struct {
		Content     []byte
		ContentType string
	}
)

func isPortInUse(port string) bool {
	addr := fmt.Sprintf(":%s", port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return false // nope
	}
	defer conn.Close()
	return true // yup
}

func serve(
	f map[string]struct {
		Content     []byte
		ContentType string
	},
	endroute string,
) {
	files = f
	initialPort := port
	for i := range 100 {
		if !isPortInUse(port) {
			break
		}
		port = fmt.Sprintf("%d", 8080+i)
		fmt.Printf("Port %s in use, trying %s...\n", initialPort, port)
		time.Sleep(10 * time.Millisecond) // Wait a bit before retrying
	}

	if isPortInUse(port) {
		panic("Could not find an available port after multiple attempts.  Last port tried:" + port)
	}
	go func() {
		time.Sleep(200 * time.Millisecond)
		fmt.Println("Server listening on port " + port)
		var url = "http://localhost:" + port + endroute
		if err := func(url string) error {
			var cmd *exec.Cmd
			switch runtime.GOOS {
			case "windows":
				cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
			case "darwin":
				cmd = exec.Command("open", url)
			default:
				cmd = exec.Command("xdg-open", url)
			}
			return cmd.Start()
		}(url); err != nil {
			panic(err)
		}
	}()
	mux := http.NewServeMux()
	mux.HandleFunc("/", memoryHandler)
	fmt.Println("Serving from memory on http://127.0.0.1:" + port + "...")
	err := http.ListenAndServe(":"+port, mux)
	if err != nil {
		panic(err)
	}
}

func memoryHandler(w http.ResponseWriter, r *http.Request) {
	var ok bool
	file := struct {
		Content     []byte
		ContentType string
	}{}
	path := strings.TrimPrefix(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	if len(parts[0]) != 64 {
		if path == "" {
			path = "/" // fallback to index
		}
		if path == "/" {
			path += "index.html"
		} else {
			path = "/" + path
		}
		fmt.Println("Path", path)
		file, ok = files[path]
		if !ok {
			http.NotFound(w, r)
			return
		}

	} else if sc = getSC(parts[0]); len(sc.VariableStringKeys) != 0 {
		endroute := strings.Join(parts[1:], "/")
		mimeType := mime.TypeByExtension(filepath.Ext(endroute))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		src := getDapp(sc)
		file, ok = src[endroute]
		if !ok {
			panic("no files here")
		}

	}

	owner := getPageOwner(sc.VariableStringKeys)
	fmt.Println(owner)
	address, err := rpc.NewAddressFromCompressedKeys([]byte(owner))
	if err != nil {
		panic(err)
	}
	address.Mainnet = false
	if !epoch.IsActive() {
		err = epoch.StartGetWork(address.String(), endpoint)
		if err != nil {
			panic(err)
		}
	}

	// // Wait for first job to be ready with a 10 second timeout
	err = epoch.JobIsReady(time.Second * 20)
	if err != nil {
		panic(err)
	}
	go func() {
		// Attempts can be called directly from the package or added to the application's API
		_, err = epoch.AttemptHashes(1000)
		if err != nil {
			panic(err)
		}
	}()

	w.Header().Set("Content-Type", file.ContentType)
	w.WriteHeader(http.StatusOK)
	w.Write(file.Content)
}
