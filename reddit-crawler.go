package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"github.com/vartanbeno/go-reddit/v2/reddit"
)
var credentials_path = "./credentials.json"

//Reads the json file containing the bots credentials for authentification in order to access the Reddit API
func SetCredentials(credentials *reddit.Credentials){
	content, err := ioutil.ReadFile(credentials_path)
	if err != nil{
		log.Fatal("Error while reading credentials", err)
	}
	err = json.Unmarshal(content, credentials)
	if err != nil{
		log.Fatal("Error during Unmarshal()", err)
	}
	//log.Printf("Username: %v, Password: %v,Id: %v, Secret: %v", credentials.Username, credentials.Password, credentials.ID, credentials.Secret)

}
func main(){
	var credentials reddit.Credentials
	SetCredentials(&credentials)
	// reddit_credentials := reddit.Credentials{ID: credentials.ID, 
	// 								Secret: credentials.Secret,
	// 								Username: credentials.Username,
	// 								Password: credentials.Password}
	//client, _ := reddit.NewClient(credentials)							
}