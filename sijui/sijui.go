package main

import (
	"crawler"
	"searchAndPrompt"
	//"sync"

)

var (
	resourcesPath = "../resources"
	redditCredentialsPath = resourcesPath+"/credentials.json"
	postAndNumberOfCommentsJsonPath = resourcesPath+"/postAndNumberOfComments.json"
	subreddit = "kenya"
	botUsername = "Sijui-bot"
	triggerWords = []string{"!sijui-bot", "sijui-bot", "!sijui", "u/sijui-bot"}
	postAndNumberOfCommentsMap = make(map[string]int)

	googleCredentialsPath = resourcesPath+"/googleCredentials.json"
	openAICredentialsPath = resourcesPath+"/openAICredentials.json"

	)


func main(){
	//Setting up the API clients
	//redditClient := crawler.SetUpRedditClient(crawler.SetRedditCredentials(&redditCredentialsPath))
	//googleService := searchAndPrompt.SetUpGoogleSearchService(searchAndPrompt.SetUpGoogleCredentials((&googleCredentialsPath)))
	//openAIClient := searchAndPrompt.SetUpOpenAIClient(searchAndPrompt.SetUpOpenAICredentials(&openAICredentialsPath))

	


}