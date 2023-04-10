package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
)
var credentials_path = "./credentials.json"
type Credentials struct{
	Username string
	Password string
	Id string
	Secret string
}

//Reads the json file containing the bots credentials for authentification in order to access the Reddit API
func (credentials *Credentials) ReadCredentials(){
	content, err := ioutil.ReadFile(credentials_path)
	if err != nil{
		log.Fatal("Error while reading credentials", err)
	}
	err = json.Unmarshal(content, credentials)
	if err != nil{
		log.Fatal("Error during Unmarshal()", err)
	}
	//log.Printf("Username: %v, Password: %v,Id: %v, Secret: %v", credentials.Username, credentials.Password, credentials.Id, credentials.Secret)

}
func main(){
	var credentials Credentials
	credentials.ReadCredentials()
}