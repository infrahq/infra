package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Welcome!\n")
}

func hello(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprintf(w, "hello, %s!\n", ps.ByName("name"))
}

func Run() {
	router := httprouter.New()
	router.GET("/", index)
	router.GET("/hello/:name", hello)

	fmt.Printf("Listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
