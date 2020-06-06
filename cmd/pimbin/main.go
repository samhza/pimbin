package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/erebid/pimbin"
	"github.com/erebid/pimbin/config"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh/terminal"
)

const usage = `usage: pimbin [-config path] <command> [options...]

	run                                 run pimbin
	create-user     <username> [hash]   create a user
	change-password <username> [hash]   change a user's password
	refresh-token   <username>          refresh a user's token
	help                                show this message`

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage)
	}
}
func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "path to configuration file")
	flag.Parse()

	var cfg *config.Server
	if configPath != "" {
		var err error
		cfg, err = config.Load(configPath)
		if err != nil {
			fmt.Printf("error loading config: %s", err.Error())
			os.Exit(1)
		}
	} else {
		cfg = config.Defaults()
	}
	db, err := pimbin.OpenSQLiteDB(cfg.DBPath)
	if err != nil {
		log.Fatalln(err)
	}

	switch cmd := flag.Arg(0); cmd {
	case "create-user":
		name := flag.Arg(1)
		input := []byte(flag.Arg(2))
		if name == "" {
			flag.Usage()
			os.Exit(1)
		}
		if len(input) == 0 {
			fmt.Printf("Password for new user %s: ", name)
			input, err = terminal.ReadPassword(int(os.Stdin.Fd()))
			fmt.Printf("\n")
			if err != nil {
				fmt.Printf("error reading password: %s\n", err)
				os.Exit(1)
			}
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
	case "change-password":
		name := flag.Arg(1)
		input := []byte(flag.Arg(2))
		if name == "" {
			flag.Usage()
			os.Exit(1)
		}
		user, err := db.GetUser(name)
		if err != nil {
			fmt.Printf("error retrieving user from db: %s\n", err)
			os.Exit(1)
		}
		if len(input) == 0 {
			fmt.Printf("New password for user %s: ", name)
			input, err = terminal.ReadPassword(int(os.Stdin.Fd()))
			fmt.Printf("\n")
			if err != nil {
				fmt.Printf("error reading password: %s\n", err)
				os.Exit(1)
			}
		}
		hash, err := bcrypt.GenerateFromPassword(input, bcrypt.DefaultCost)
		if err != nil {
			fmt.Printf("error hashing password: %s\n", err)
			os.Exit(1)
		}
		user.Password = string(hash)
		err = db.UpdatePassword(user)
		if err != nil {
			fmt.Printf("error updating token in db: %s\n", err)
			os.Exit(1)
		}
	case "refresh-token":
		name := flag.Arg(1)
		if name == "" {
			flag.Usage()
			os.Exit(1)
		}
		user, err := db.GetUser(name)
		if err != nil {
			fmt.Printf("error getting user from db: %s\n", err)
			os.Exit(1)
		}
		token, err := db.RefreshToken(user)
		if err != nil {
			fmt.Printf("error inserting token into db: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("%s's token: %s\n", name, token)
	case "run":
		s, err := pimbin.NewServer(db)
		if err != nil {
			log.Fatalln(err)
		}
		s.UploadsDir = cfg.UploadsDir
		s.BaseURL = cfg.BaseURL
		s.Filter = cfg.Filter
		s.FilterAllow = cfg.FilterAllow
		s.MaxBodySize = cfg.MaxBodySize
		s.CSSPath = cfg.CSSPath
		s.SiteName = cfg.SiteName
		log.Fatalln(http.ListenAndServe(cfg.Addr, s))
	default:
		flag.Usage()
		if cmd != "help" {
			os.Exit(1)
		}
	}
}
