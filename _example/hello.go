package main

import (
	"io"
	"net/http"
	"os"
)

func main() {
	http.ListenAndServe(":"+os.Getenv("PORT"), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "Hello world\n")
	}))
}
