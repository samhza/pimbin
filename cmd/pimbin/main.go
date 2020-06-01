package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/erebid/pimbin"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	// TODO config file
	db, err := pimbin.OpenSQLiteDB("./pimbin.db")
	if err != nil {
		log.Fatalln(err)
	}
	flag.Parse()
	switch cmd := flag.Arg(0); cmd {
	case "create-user":
		name := flag.Arg(1)
		if name == "" {
			println("provide a username")
			os.Exit(1)
		} else if len(name) > 255 {
			println("username longer than 255 characters")
			os.Exit(1)
		}
		fmt.Printf("Password for new user %s: ", name)
		input, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		fmt.Printf("\n")
		if err != nil {
			fmt.Printf("error reading password: %s\n", err)
			os.Exit(1)
		}
		hash, err := bcrypt.GenerateFromPassword(input, bcrypt.DefaultCost)
		if err != nil {
			fmt.Printf("error hashing password: %s\n", err)
			os.Exit(1)
		}
		user := &pimbin.User{
			Name:     name,
			Password: string(hash),
		}
		err = db.CreateUser(user)
		if err != nil {
			fmt.Printf("error inserting user into db: %s\n", err)
			os.Exit(1)
		}
		token, err := db.RefreshToken(user)
		if err != nil {
			fmt.Printf("error inserting token into db: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("%s's token: %s\n", name, token)
	default:
		s, err := pimbin.NewServer(db)
		if err != nil {
			log.Fatalln(err)
		}
		s.UploadsDir = "./uploads"
		s.BaseURL = "http://localhost:8080/"
		log.Fatalln(http.ListenAndServe(":8080", s))
	}
}
