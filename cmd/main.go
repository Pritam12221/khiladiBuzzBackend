package main

import (
	db "khiladiBuzz/database"
	"log"
	"os"
	s "khiladiBuzz/routes"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load("../.env")
	if err != nil {
		log.Println(".env file not found")
	}
	err = db.ConnectAndMigrate(
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		db.SSLMode(os.Getenv("DB_SSLMODE")),
	)
	if err != nil {
		log.Fatal(err)
	}

 srv:=s.ServerRoutes();
 srv.Run();	
}
