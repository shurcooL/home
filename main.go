// home is my personal website.
package main

import (
	"flag"
	"log"
	"net/http"
	"os/user"
	"path/filepath"
)

var httpFlag = flag.String("http", ":8080", "Listen for HTTP connections on this address.")

func main() {
	flag.Parse()

	user, err := user.Current()
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Started.")

	http.Handle("/robots.txt", http.NotFoundHandler())
	http.Handle("/", http.FileServer(http.Dir(filepath.Join(user.HomeDir, "Dropbox", "Public", "dmitri"))))

	err = http.ListenAndServe(*httpFlag, nil)
	if err != nil {
		log.Fatalln(err)
	}
}
