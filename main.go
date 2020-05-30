package main

import (
	"log"
	"net/http"
)

func main() {
	db, err := OpenSQLiteDB("./pimbin.db")
	if err != nil {
		log.Fatalln(err)
	}
	s := NewServer(db)
	s.UploadsDir = "./uploads"
	http.ListenAndServe(":8080", s)
}
