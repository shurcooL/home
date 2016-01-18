// home is my personal website.
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

var (
	httpFlag = flag.String("http", ":8080", "Listen for HTTP connections on this address.")
	blogFlag = flag.String("blog", "", "Path to wordpress blog XML file.")
)

func main() {
	flag.Parse()

	var err error
	err = initBlog(*blogFlag)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Started.")

	http.Handle("/robots.txt", http.NotFoundHandler())
	http.Handle("/", http.FileServer(http.Dir(filepath.Join(os.Getenv("HOME"), "Dropbox", "Public", "dmitri"))))

	err = http.ListenAndServe(*httpFlag, nil)
	if err != nil {
		log.Fatalln(err)
	}
}
