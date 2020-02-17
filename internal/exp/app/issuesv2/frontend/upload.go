package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gopherjs/gopherjs/js"
	"honnef.co/go/js/dom"
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

	nameFunc, err := plainTextString(items)
	if err != nil {
		nameFunc = func() string { return "Image" }
	}

	go func() {
		b := blobToBytes(file)
		name := nameFunc()

		resp, err := http.Post("/api/usercontent", "image/png", bytes.NewReader(b))
		if err != nil {
			log.Println(err)
			return
		}
		defer resp.Body.Close()
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
	value, start, end := t.Value, t.SelectionStart, t.SelectionEnd
	t.Value = value[:start] + inserted + value[end:]
	t.SelectionStart, t.SelectionEnd = start+len(inserted), start+len(inserted)
}

// imagePNGFile tries to get an "image/png" file from items.
func imagePNGFile(items *js.Object) (file *js.Object, err error) {
	for i := 0; i < items.Length(); i++ {
		item := items.Index(i)
		if item.Get("kind").String() != "file" || item.Get("type").String() != "image/png" {
			continue
		}
		return item.Call("getAsFile"), nil
	}
	return nil, fmt.Errorf("not found")
}

// plainTextString tries to get a "text/plain" string from items.
// The returned func blocks until the string is available.
func plainTextString(items *js.Object) (func() string, error) {
	for i := 0; i < items.Length(); i++ {
		item := items.Index(i)
		if item.Get("kind").String() != "string" || item.Get("type").String() != "text/plain" {
			continue
		}
		s := make(chan string)
		item.Call("getAsString", func(o *js.Object) {
			go func() { s <- o.String() }()
		})
		return func() string { return <-s }, nil
	}
	return nil, fmt.Errorf("not found")
}

// blobToBytes converts a Blob to []byte.
func blobToBytes(blob *js.Object) []byte {
	b := make(chan []byte)
	fileReader := js.Global.Get("FileReader").New()
	fileReader.Set("onload", func() {
		b <- js.Global.Get("Uint8Array").New(fileReader.Get("result")).Interface().([]byte)
	})
	fileReader.Call("readAsArrayBuffer", blob)
	return <-b
}
