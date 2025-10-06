package main

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

var node = `http://127.0.0.1:20000/json_rpc`

func main() {

	fmt.Println("the purpose of this experiment is to use a scid to serve a website")
	// "sim://<SCID>"
	// submit sURL/SCID
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
			sc := getSC(host)
			// gather data
			data := getData(sc.VariableStringKeys, host)
			fmt.Println(data)
			// construct files
			files := getDapp(sc)
			// serve content
			serve(files)
		}()

		// fun(files)
		// open browser

	}
	w.Resize(fyne.NewSize(300, entry.MinSize().Height))
	w.ShowAndRun()
}
