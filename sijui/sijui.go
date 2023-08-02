package main

import (
	"crawler"
	"fmt"
	"log"

	"searchAndPrompt"
	"time"
)

var (
	resourcesPath                   = "../resources"
	redditCredentialsPath           = resourcesPath + "/redditCredentials.json"
	postAndNumberOfCommentsJsonPath = resourcesPath + "/postAndNumberOfComments.json"
	subreddit                       = "Sijui"
	botUsername                     = "Sijui-bot"
	triggerWords                    = []string{"!sijui-bot", "!sijui", "u/sijui-bot"}

	googleCredentialsPath = resourcesPath + "/googleCredentials.json"

	redditClient               = crawler.SetUpRedditClient(crawler.SetRedditCredentials(&redditCredentialsPath))
	postService                = redditClient.Post
	commentService             = redditClient.Comment
	googleService              = searchAndPrompt.SetUpGoogleSearchService(searchAndPrompt.SetUpGoogleCredentials((&googleCredentialsPath)))
	postAndNumberOfCommentsMap = crawler.CreatePostsNumberOfCommentsMapFromJson(&postAndNumberOfCommentsJsonPath)
)

const (
	maxGoogleSearchQueryQuota = 100
	internetTestDelay = 10 * time.Second
	googleRetryDelay  = 5 * time.Second
	redditRetryDelay  = 10 * time.Second
	retryCount        = 10
)

func createBotReply(botName *string, googleResults *[]searchAndPrompt.GoogleResult) *string {
	var resultString string
	for _, result := range *(googleResults) {
		resultString += fmt.Sprintf("[%s](%s): %s\n\n", result.Title, result.Link, result.Snippet)
	}
	botReply := fmt.Sprintf("%s here! I've found the following information that might help answer your question:\n\n%s\n\nI hope this helps! Let me know if you have any other questions.", *botName, resultString)
	return &botReply
}

func main() {
	// Fetch new and top posts
	log.Println("Fetching new and top posts.. .")
	newPosts, _, err := crawler.FetchNewPosts(redditClient, &subreddit)
	if err != nil {
		log.Fatal("Error fetching new posts: ", err)
	}
	topPosts, _, err := crawler.FetchTopPosts(redditClient, &subreddit)
	if err != nil {
		log.Fatal("Error fetching top posts: ", err)
	}
	// Combine topPosts and newPosts into one slice
	// Unpack newPosts first since append only accepts elements not arrays
	posts := append((*topPosts), (*newPosts)...)
	// Check if the fetched posts have new comments since the last check then update the map accordingly, returns only the posts that have new comments
	posts = crawler.FindPostsThatHaveHaveNewComments(postAndNumberOfCommentsMap, &posts)
	// Update the postsNumberofCommentsJSON with the new map
	crawler.UpdateJSONWithPostsNumberOfCommentsMap(postAndNumberOfCommentsMap, &postAndNumberOfCommentsJsonPath)
	// Returns the post and it's comments
	log.Printf("Finding comments to %v posts that have new comments.. .", len(posts))
	postsAndComments := crawler.FindPostsCommentsScheduler(&posts, postService)
	// Find and return in comments that have trigger words, returns a map that has the form {commentID: question}
	queriedComments := crawler.CheckTriggerWordScheduler(&botUsername, &triggerWords, postsAndComments)
	for commentID, question := range *queriedComments {
		var googleResults *[]searchAndPrompt.GoogleResult
		localRetryCount := retryCount
		for localRetryCount > 0 {
			googleResults, err = searchAndPrompt.GoogleSearch(&question, googleService)
			if err != nil {
				time.Sleep(googleRetryDelay)
				localRetryCount -= 1
			} else {
				localRetryCount = retryCount
				break
			}
		}
		botReply := createBotReply(&botUsername, googleResults)
		log.Println("Replying to comment.. .")
		crawler.Reply(&commentID, botReply, commentService)
	}

}