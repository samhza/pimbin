package main

import (
	"log"
	"net/http"
)

func main() {
	// TODO: config file
	db, err := OpenSQLiteDB("./pimbin.db")
	if err != nil {
		log.Fatalln(err)
	}
	s := NewServer(db)
	s.UploadsDir = "./uploads"
	s.BaseURL = "http://localhost:8080/"
	http.ListenAndServe(":8080", s)
}
