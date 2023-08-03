package main

import (
	"crawler"
	"fmt"
	"os"
	"path/filepath"
	"searchAndPrompt"
	"shared"
	"strconv"
	"time"

	"github.com/vartanbeno/go-reddit/v2/reddit"
)

var (
	baseDirectoryPath               = "../"
	resourcesPath                   = filepath.Join(baseDirectoryPath, "resources")
	redditCredentialsPath           = filepath.Join(resourcesPath, "redditCredentials.json")
	postAndNumberOfCommentsJsonPath = filepath.Join(resourcesPath, "postAndNumberOfComments.json")
	googleCredentialsPath           = filepath.Join(resourcesPath, "googleCredentials.json")
	currentQueryCountPath           = filepath.Join(resourcesPath, "currentQueryCount.txt")
	startTimePath                   = filepath.Join(resourcesPath, "startTime.txt")
	totalQuestionsAnsweredPath      = filepath.Join(resourcesPath, "totalAnswered.txt")
	subreddit                       = "Kenya"
	botUsername                     = "Sijui-bot"
	// The trigger words are in lowercase cause when we check for them in the comment body, the comment body is converted to loweecase first
	triggerWords = []string{"!sijui-bot", "!sijui", "u/sijui-bot"}

	currentQueryCount, _       = strconv.ParseUint(readPreviousStoredValue(currentQueryCountPath, "0"), 10, 8)
	startTime, _               = strconv.ParseInt(readPreviousStoredValue(startTimePath, fmt.Sprintf("%v", time.Now().Unix())), 10, 64)
	totalAnswered, _           = strconv.ParseUint(readPreviousStoredValue(totalQuestionsAnsweredPath, "0"), 10, 64)
	redditClient               = crawler.SetUpRedditClient(crawler.SetRedditCredentials(redditCredentialsPath))
	postService                = redditClient.Post
	commentService             = redditClient.Comment
	googleService              = searchAndPrompt.SetUpGoogleSearchService(searchAndPrompt.SetUpGoogleCredentials((googleCredentialsPath)))
	postAndNumberOfCommentsMap = crawler.CreatePostsNumberOfCommentsMapFromJson(postAndNumberOfCommentsJsonPath)
)

const (
	maxGoogleSearchQueryQuota = 100
	redditSleepDelay          = 60 * time.Second
	// The extra hour is handling for just in case something goes wrong
	dayInSeconds = 25 * 60 * 60
)

func readTextFromFile(path string) string {
	readBytes, err := os.ReadFile(path)
	if err != nil {
		shared.LogError(fmt.Sprintf("Error reading text from this file %v", path), err)
	}
	return string(readBytes)
}

func writeTextToFile(path string, text string) {
	file, err := os.OpenFile(path, os.O_RDWR, 0666)
	if err != nil {
		shared.LogError(fmt.Sprintf("Error opening this file %v to write text into it", path), err)
	}
	defer file.Close()
	_, err = file.WriteString(text)
	if err != nil {
		shared.LogError(fmt.Sprintf("Error writing text to this file %v", path), err)
	}
}

func createBotReply(botName string, googleResults *[]searchAndPrompt.GoogleResult) string {
	var resultString string
	for _, result := range *(googleResults) {
		resultString += fmt.Sprintf("[%s](%s): %s\n\n", result.Title, result.Link, result.Snippet)
	}
	botReply := fmt.Sprintf("%s here! I've found the following information that might help answer your question:\n\n%s\n\nI hope this helps! Let me know if you have any other questions.", botName, resultString)
	return botReply
}

func readPreviousStoredValue(path string, defaultValue string) string {
	if _, err := os.Stat(path); err != nil {
		currentQueryCountFile, err := os.Create(path)
		if err != nil {
			shared.LogError("Error creating currentQueryCount.txt: ", err)
		}
		defer currentQueryCountFile.Close()
		writeTextToFile(path, defaultValue)
	}
	return readTextFromFile(path)

}

func fetchQueriedComments() *map[string]string {
	// Fetch top and new posts
	fetchTopPosts := func() (interface{}, error) {
		topPosts, _, err := crawler.FetchTopPosts(redditClient, subreddit)
		return topPosts, err
	}
	fetchNewPosts := func() (interface{}, error) {
		newPosts, _, err := crawler.FetchNewPosts(redditClient, subreddit)
		return newPosts, err
	}
	shared.LogInfo("Fetching top posts.. .")
	topPosts := shared.RetryOnErrorWrapper(fetchTopPosts, "Error fetching top posts").(*[]*reddit.Post)
	shared.LogInfo("Fetching new posts.. .")
	newPosts := shared.RetryOnErrorWrapper(fetchNewPosts, "Error fetching new posts").(*[]*reddit.Post)
	// Combine topPosts and newPosts into one slice
	// Unpack newPosts first since append only accepts elements not arrays
	posts := append((*topPosts), (*newPosts)...)
	// Check if the fetched posts have new comments since the last check then update the map accordingly, returns only the posts that have new comments
	posts = crawler.FindPostsThatHaveHaveNewComments(postAndNumberOfCommentsMap, &posts)
	// Update the postsNumberofCommentsJSON with the new map
	crawler.UpdateJSONWithPostsNumberOfCommentsMap(postAndNumberOfCommentsMap, postAndNumberOfCommentsJsonPath)
	// Returns the post and it's comments
	shared.LogInfo(fmt.Sprintf("Finding comments to %v posts that have new comments.. .", len(posts)))
	postsAndComments := crawler.FindPostsCommentsScheduler(&posts, postService)
	// Find and return in comments that have trigger words, returns a map that has the form {commentID: question}
	queriedComments := crawler.CheckTriggerWordScheduler(botUsername, &triggerWords, postsAndComments)
	return queriedComments
}

func replyToQueriedComments(queriedComments *map[string]string) {
	for commentID, question := range *queriedComments {
		anon := func() (interface{}, error) {
			return searchAndPrompt.GoogleSearch(question, googleService)
		}
		googleResults, _ := shared.RetryOnErrorWrapper(anon, "Error getting google search result").(*[]searchAndPrompt.GoogleResult)
		botReply := createBotReply(botUsername, googleResults)
		shared.LogInfo("Replying to comment.. .")
		crawler.Reply(commentID, botReply, commentService)
		currentQueryCount += 1
		totalAnswered += 1
		writeTextToFile(totalQuestionsAnsweredPath, fmt.Sprintf("%v", totalAnswered))
		break
	}
}

func main() {
	shared.LogInfo(fmt.Sprintf("%v, waking up!!!", botUsername))
	for {
		for currentQueryCount < maxGoogleSearchQueryQuota {
			queriedComments := fetchQueriedComments()
			replyToQueriedComments(queriedComments)
			writeTextToFile(currentQueryCountPath, fmt.Sprintf("%v", currentQueryCount))
			shared.LogInfo(fmt.Sprintf("Sleeping for %v .. .", redditSleepDelay))
			time.Sleep(redditSleepDelay)
		}
		elapsedTime := time.Now().Unix() - startTime
		remainingTimeTillFullDay := dayInSeconds - elapsedTime
		if remainingTimeTillFullDay > 0 {
			// we have to convert the seconds to nanoseconds
			sleepDuration := time.Duration(remainingTimeTillFullDay * 1000 * 1000 * 1000)
			shared.LogInfo(fmt.Sprintf("Sleeping for %v", sleepDuration))
			time.Sleep(sleepDuration)

		}
		currentQueryCount = 0
		writeTextToFile(currentQueryCountPath, "0")
		startTime = time.Now().Unix()
		writeTextToFile(startTimePath, fmt.Sprintf("%v", startTime))
		shared.LogInfo("Starting new cycle. Yay!!!")
	}
}
