package main

import (
	"crawler"
	"searchAndPrompt"
	"github.com/vartanbeno/go-reddit/v2/reddit"

)

var (
	credentialsPath = "./credentials.json"
	postAndNumberOfCommentsJsonPath = "./posts_comment_count.json"
	subreddit = "kenya"
	botUsername = "Sijui-bot"
	//postOptions = reddit.ListPostOptions{ ListOptions: reddit.ListOptions{Limit: 100}, Time:"day"}
	postOptions = reddit.ListOptions{Limit: 100}
	triggerWords = []string{"!sijui-bot", "sijui-bot", "!sijui", "u/sijui-bot"}
	postAndNumberOfCommentsMap = make(map[string]int)

	googleCredentialsPath = "../googleCredentials.json"
	openAICredentialsPath = "../openAICredentials.json"

	)


func main(){
}