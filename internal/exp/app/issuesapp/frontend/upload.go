// +build js,wasm,go1.14

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"syscall/js"

	"honnef.co/go/js/dom/v2"
)

func PasteHandler(e dom.Event) {
	ce := e.(*dom.ClipboardEvent)

	items := ce.Get("clipboardData").Get("items")
	file, err := imagePNGFile(items)
	if err != nil {
		// No image to paste.
		return
	}

	// From this point, we're taking on the responsibility to handle this clipboard event.
	ce.PreventDefault()

	nameFn, err := plainTextString(items)
	if err != nil {
		nameFn = func() string { return "Image" }
	}

	go func() {
		b, name := blobToBytes(file), nameFn()

		resp, err := http.Post("/api/usercontent", "image/png", bytes.NewReader(b))
		if err != nil {
			log.Println(err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := ioutil.ReadAll(resp.Body)
			log.Println(fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body))
			return
		} else if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
			log.Println(fmt.Errorf("got Content-Type %q, want %q", ct, "application/json"))
			return
		}
		var uploadResponse struct {
			URL   string
			Error string
		}
		err = json.NewDecoder(resp.Body).Decode(&uploadResponse)
		if err != nil {
			log.Println(err)
			return
		}
		if uploadResponse.Error != "" {
			log.Println(uploadResponse.Error)
			return
		}

		insertText(ce.Target().(*dom.HTMLTextAreaElement), fmt.Sprintf("![%s](%s)\n\n", name, uploadResponse.URL))
	}()
}

func insertText(t *dom.HTMLTextAreaElement, inserted string) {
	value, start, end := t.Value(), t.SelectionStart(), t.SelectionEnd()
	t.SetValue(value[:start] + inserted + value[end:])
	t.SetSelectionStart(start + len(inserted))
	t.SetSelectionEnd(start + len(inserted))
}

// imagePNGFile tries to get an "image/png" file from items.
func imagePNGFile(items js.Value) (file js.Value, _ error) {
	for i := 0; i < items.Length(); i++ {
		item := items.Index(i)
		if item.Get("kind").String() != "file" || item.Get("type").String() != "image/png" {
			continue
		}
		return item.Call("getAsFile"), nil
	}
	return js.Value{}, os.ErrNotExist
}

// plainTextString tries to get a "text/plain" string from items.
// The returned func blocks until the string is available.
func plainTextString(items js.Value) (func() string, error) {
	for i := 0; i < items.Length(); i++ {
		item := items.Index(i)
		if item.Get("kind").String() != "string" || item.Get("type").String() != "text/plain" {
			continue
		}
		ch := make(chan string, 1)
		var f js.Func
		f = js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
			ch <- args[0].String()
			f.Release()
			return nil
		})
		item.Call("getAsString", f)
		return func() string { return <-ch }, nil
	}
	return nil, os.ErrNotExist
}

// blobToBytes converts a Blob to a []byte.
func blobToBytes(blob js.Value) []byte {
	ch := make(chan []byte)
	fileReader := js.Global().Get("FileReader").New()
	f := js.FuncOf(func(js.Value, []js.Value) interface{} {
		r := js.Global().Get("Uint8Array").New(fileReader.Get("result"))
		b := make([]byte, r.Length())
		js.CopyBytesToGo(b, r)
		ch <- b
		return nil
	})
	defer f.Release()
	fileReader.Set("onload", f)
	fileReader.Call("readAsArrayBuffer", blob)
	return <-ch
}
