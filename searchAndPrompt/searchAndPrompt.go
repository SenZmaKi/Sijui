package main

import (
//	"context"
	"encoding/json"
	"log"
	"os"

//	"encoding.json"
//	"google.golang.org/api/customsearch/v1"
)

var (
	googleCredentialsPath = "./googleCredentials.json"
)

//unrepliedComments *map[string]string
//                    commentID  question


func SetUpGoogleCredentials(credentialsPath *string)*map[string]string{
	googleCredentials := make(map[string]string)
	if file, err := os.Open(*credentialsPath); err!=nil{
		log.Fatal("Error reading googleCredentials.json ", err)
	}else{
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&googleCredentials); err!=nil{
			log.Fatal("Error decoding googleCredentials.json into googleCredentials map", err)
		}
	}
	return &googleCredentials
}

func main(){
	googleCredentials := SetUpGoogleCredentials(&googleCredentialsPath)
	log.Print(googleCredentials)
}