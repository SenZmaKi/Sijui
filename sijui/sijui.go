package main

import (
	"crawler"
	"searchAndPrompt"

)

var (
	redditCredentialsPath = "../resources/credentials.json"
	postAndNumberOfCommentsJsonPath = "../resources/postAndNumberOfComments.json"
	subreddit = "kenya"
	botUsername = "Sijui-bot"
	triggerWords = []string{"!sijui-bot", "sijui-bot", "!sijui", "u/sijui-bot"}
	postAndNumberOfCommentsMap = make(map[string]int)

	googleCredentialsPath = "../resources/googleCredentials.json"
	openAICredentialsPath = "../resources/openAICredentials.json"

	)


func main(){
}