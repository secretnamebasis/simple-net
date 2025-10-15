package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/civilware/epoch"
	"github.com/deroproject/derohe/globals"
	"github.com/deroproject/derohe/rpc"
)

var (
	protocol = `http://`
	endpoint = "127.0.0.1:20000"
	node     = protocol + endpoint
	json_rpc = node + `/json_rpc`
	sc       rpc.GetSC_Result
	dev      = "deto1qyvyeyzrcm2fzf6kyq7egkes2ufgny5xn77y6typhfx9s7w3mvyd5qqynr5hx"
)

func main() {
	defer epoch.StopGetWork()

	fmt.Println("the purpose of this experiment is to use a scid to serve a website")
	// "sim://<SCID>"
	// submit sURL/SCID
	globals.Arguments["--testnet"] = true
	globals.Arguments["--simulator"] = true
	globals.Arguments["--daemon-address"] = endpoint
	globals.InitNetwork()

	address, err := rpc.NewAddress(dev)
	if err != nil {
		panic(err)
	}
	fmt.Println()
	address.Mainnet = false
	epoch.SetMaxThreads(runtime.GOMAXPROCS(0))

	err = epoch.StartGetWork(address.String(), endpoint)
	if err != nil {
		panic(err)
	}
	// // Wait for first job to be ready with a 20 second timeout
	err = epoch.JobIsReady(time.Second * 20)
	if err != nil {
		panic(err)
	}
	go func() {

		// // Attempts can be called directly from the package or added to the application's API
		_, err := epoch.AttemptHashes(1000)
		if err != nil {
			panic(err)
		}
	}()

	a := app.New()
	w := a.NewWindow("simple-internet")
	entry := widget.NewEntry()
	w.SetContent(container.NewAdaptiveGrid(1,
		container.NewVBox(
			layout.NewSpacer(),
			entry,
		)))
	// resolve submission
	entry.OnSubmitted = func(s string) {
		go func() {
			// resolve scheme
			var sURL string = s
			if strings.Contains(sURL, `sim://`) {
				sURL = strings.TrimPrefix(sURL, `sim://`)
			}
			parts := strings.Split(sURL, "/")
			host := strings.ToLower(parts[0])
			endroute := "/" + strings.Join(parts[1:], "/")
			fmt.Println(host, endroute)
			// validate scid
			sc = getSC(host)

			redirects := 0
			max := 100
		redirect:

			if sc.Code == "" {
				dialog.ShowError(errors.New("code is empty"), w)
				return
			}
			// fmt.Println(sc)
			// gather data
			keys := sc.VariableStringKeys
			// fmt.Println(keys)
			a_id, ok := keys["account"].(string)
			if !ok {
				panic("no account id")
			}
			if a_id == "" {
				panic("account id cannot be empty")
			}
			// fmt.Println(a_id)
			b, err := hex.DecodeString(a_id)
			if err != nil {
				panic(err)
			}
			account_id := string(b)
			account := getSC(account_id)
			// fmt.Println(sc)
			d := getData(account.VariableStringKeys, host)
			var data map[string]any
			if err := json.Unmarshal([]byte(d), &data); err != nil {
				panic(err)
			}
			status := data["Status"].(float64)
			switch status {
			case http.StatusOK:
			case http.StatusTemporaryRedirect:
				redirects++
				sc = getSC(data["Redirect"].(string))
				if max <= redirects {
					dialog.ShowError(errors.New("max number of redirects"), w)
					return
				}
				goto redirect
			case http.StatusNoContent:
				fallthrough
			default:
				dialog.ShowError(errors.New("site is not listed"), w)
				return
			}

			owner := getPageOwner(account.VariableStringKeys)
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
			if err = epoch.SetAddress(address.String()); err != nil {
				panic(err)
			}
			go func() {

				// Attempts can be called directly from the package or added to the application's API
				_, err = epoch.AttemptHashes(1000)
				if err != nil {
					panic(err)
				}
			}()
			// construct files
			files := getDapp(sc)

			// serve content
			serve(files, endroute)

		}()
	}
	// fun(files)
	// open browser

	w.Resize(fyne.NewSize(550, entry.MinSize().Height))
	w.ShowAndRun()
}
