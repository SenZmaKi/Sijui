package main

import (
	"crawler"
	"searchAndPrompt"
	"log"
	"fmt"
	//"sync"

)

var (
	resourcesPath = "../resources"
	redditCredentialsPath = resourcesPath+"/redditCredentials.json"
	postAndNumberOfCommentsJsonPath = resourcesPath+"/postAndNumberOfComments.json"
	subreddit = "KindaLooksLikeaDildo"
	botUsername = "Sijui-bot"
	triggerWords = []string{"!sijui-bot", "!sijui", "u/sijui-bot"}
	postAndNumberOfCommentsMap = make(map[string]int)

	googleCredentialsPath = resourcesPath+"/googleCredentials.json"
	openAICredentialsPath = resourcesPath+"/openAICredentials.json"

	)

func createBotReply(botName *string, promptResponse *string, googleResults *[]searchAndPrompt.GoogleResult) *string{
	var resultString string
	for _, result := range *(googleResults){
		resultString += fmt.Sprintf("[%s](%s): %s\n\n", result.Title, result.Link, result.Snippet)
	}
	botReply := fmt.Sprintf("%s here! I've found the following information that might help answer your question:\n\n%s\n\nChat GPT says:\n\n%s\n\nI hope this helps! Let me know if you have any other questions.", *botName, resultString, *promptResponse)
	return &botReply
}

func main(){
	//Setting up the API clients and services
	redditClient := crawler.SetUpRedditClient(crawler.SetRedditCredentials(&redditCredentialsPath))
	postService := redditClient.Post
	commentService := redditClient.Comment
	googleService := searchAndPrompt.SetUpGoogleSearchService(searchAndPrompt.SetUpGoogleCredentials((&googleCredentialsPath)))
	openAIClient := searchAndPrompt.SetUpOpenAIClient(searchAndPrompt.SetUpOpenAICredentials(&openAICredentialsPath))

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
	postsAndComments := crawler.FindPostsCommentsScheduler(&posts, postService)
	//Find and return in comments that have trigger words, returns a map that has the form {commentID: question}
	queriedComments := crawler.CheckTriggerWordScheduler(&botUsername, &triggerWords, postsAndComments)
	

	for commentID, question := range *queriedComments{
		googleResults := searchAndPrompt.GoogleSearch(&question, googleService)
		promptResponse := searchAndPrompt.PromptGpt(openAIClient, &question)
		botReply := createBotReply(&botUsername, promptResponse, googleResults)
		crawler.Reply(&commentID, botReply, commentService)
	}

	}




	
	
