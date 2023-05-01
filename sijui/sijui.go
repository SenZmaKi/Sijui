package main

import (
	"crawler"
	//"searchAndPrompt"
	"log"
	//"sync"

)

var (
	resourcesPath = "../resources"
	redditCredentialsPath = resourcesPath+"/redditCredentials.json"
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
	redditClient := crawler.SetUpRedditClient(crawler.SetRedditCredentials(&redditCredentialsPath))
	//googleService := searchAndPrompt.SetUpGoogleSearchService(searchAndPrompt.SetUpGoogleCredentials((&googleCredentialsPath)))
	//openAIClient := searchAndPrompt.SetUpOpenAIClient(searchAndPrompt.SetUpOpenAICredentials(&openAICredentialsPath))

	//Check if the postsNumberOFCOmmentsJson file exits, if not create it
	if !crawler.CheckIfPostsNumberOfCommentsJSONExists(&postAndNumberOfCommentsJsonPath){crawler.CreatePostsNumberOfCommentsJSON(&postAndNumberOfCommentsJsonPath)}
	//Fetch new and top posts
	newPosts, _, err := crawler.FetchNewPosts(redditClient, &subreddit)
	if err != nil{log.Fatal("Error fetching new posts ", err)}
	topPosts, _, err := crawler.FetchTopPosts(redditClient, &subreddit)
	if err != nil{log.Fatal("Error fetching top posts ", err)}
	//Combine topPosts and newPosts into one slice
	//Unpack newPosts first since append only accepts elements not arrays
	posts := append((*topPosts), (*newPosts)...)
	//Write the content in the postsNumberOfCommentJson to the map
	crawler.WriteJsonToPostsNumberOFCommentsMap(&postAndNumberOfCommentsMap, &postAndNumberOfCommentsJsonPath)
	//Check if the fetched posts have new comments since the last check then update the map accordingly, returns only the posts that have new comments
	posts = crawler.FindPostsThatHaveHaveNewComments(&postAndNumberOfCommentsMap, &posts)
	//Update the postsNumberofCommentsJSON with the new map
	crawler.UpdateJSONWithPostsNumberOfCommentsMap(&postAndNumberOfCommentsMap, &postAndNumberOfCommentsJsonPath)
	//Returns the post and it's comments
	postsAndComments := crawler.FindPostsCommentsScheduler(&posts, redditClient.Post)
	//Find and return in comments that have trigger words, returns a map that has the form {commentID: question}
	queriedComments := crawler.CheckTriggerWordScheduler(&botUsername, &triggerWords, postsAndComments)

	
	}




	
	
