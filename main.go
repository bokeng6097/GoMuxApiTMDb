// main.go

package main

import (
	"encoding/json"
	"log"
	"os"
)

type Config struct {
	User     string
	Password string
	Dbname   string
}

func main() {
	a := App{}

	file, _ := os.Open("conf.json")
	defer file.Close()
	decoder := json.NewDecoder(file)
	config := Config{}
	err := decoder.Decode(&config)
	if err != nil {
		log.Fatal(err)
	}

	a.Initialize(config.User, config.Password, config.Dbname)

	a.Run(":8080")
}
