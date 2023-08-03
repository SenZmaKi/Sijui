package crawler

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"

	"shared"

	"github.com/vartanbeno/go-reddit/v2/reddit"
)

// Reads the json file containing the bots credentials for authentification in order to access the Reddit API
func SetRedditCredentials(credentialsPath string) *reddit.Credentials {
	credentials := &reddit.Credentials{}
	if file, err := os.Open(credentialsPath); err != nil {
		shared.LogError("Error opening redditCredentials.json", err)
	} else {
		defer file.Close()
		if err := json.NewDecoder(file).Decode(credentials); err != nil {
			shared.LogError("Error decoding redditCredentials.json into redditCredentials map", err)
		}
	}
	return credentials

}

// Sets up reddit API client using the provided credentials
func SetUpRedditClient(credentials *reddit.Credentials) *reddit.Client {
	client, err := reddit.NewClient(*credentials)
	if err != nil {
		shared.LogError("Error while setting up reddit client", err)
	}
	return client

}

func CheckIfPostsNumberOfCommentsJSONExists(postAndNumberOfCommentsJsonPath string) bool {
	if _, err := os.Stat(postAndNumberOfCommentsJsonPath); err == nil {
		return true
	} else {
		return false
	}
}

func CreatePostsNumberOfCommentsJSON(postAndNumberOfCommentsJsonPath string) {
	if file, err := os.Create(postAndNumberOfCommentsJsonPath); err != nil {
		shared.LogError("Error creating the postsAndNumberOfComments.json file", err)
	} else {
		file.Close()
	}
}

func UpdateJSONWithPostsNumberOfCommentsMap(postAndNumberOfCommentsMap *map[string]int, postAndNumberOfCommentsJsonPath string) {
	// 0666 is a flag that allows for Read and Write permissions
	if file, err := os.Create(postAndNumberOfCommentsJsonPath); err != nil {
		shared.LogError("Error opening JSON in attempt to update it with the new(changed) post and number of comments read from map: ", err)
	} else {
		defer file.Close()
		encoder := json.NewEncoder(file)
		if err := encoder.Encode(postAndNumberOfCommentsMap); err != nil {
			shared.LogError("Error updating JSON with the new post and number of comments read from map: ", err)
		}
	}

}

func CreatePostsNumberOfCommentsMapFromJson(postAndNumberOfCommentsJsonPath string) *map[string]int {
	postsNumberOfCommentsMap := make(map[string]int)
	if !CheckIfPostsNumberOfCommentsJSONExists(postAndNumberOfCommentsJsonPath) {
		CreatePostsNumberOfCommentsJSON(postAndNumberOfCommentsJsonPath)
	}
	if file, err := os.Open(postAndNumberOfCommentsJsonPath); err != nil {
		shared.LogError("Error while opening the JSON file in an attempt to write JSON to map", err)
	} else {
		defer file.Close()
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&postsNumberOfCommentsMap); err != nil {
			shared.LogError("Error writing values read from JSON to map", err)
		}
	}
	return &postsNumberOfCommentsMap
}

func FindPostsThatHaveHaveNewComments(postAndNumberOfCommentsMap *map[string]int, posts *[]*reddit.Post) []*reddit.Post {
	changedPosts := make([]*reddit.Post, 0, len(*posts)/4)
	accessedKeys := make(map[string]bool)
	for _, post := range *posts {
		// update the accessed keys
		accessedKeys[post.FullID] = true
		// If we have the post in our map
		if _, ok := (*postAndNumberOfCommentsMap)[post.FullID]; ok {
			// If the number of comments on the post have changed
			if post.NumberOfComments != ((*postAndNumberOfCommentsMap)[post.FullID]) {
				(*postAndNumberOfCommentsMap)[post.FullID] = post.NumberOfComments
				// add the post to the posts to be checked for the trigger word
				changedPosts = append(changedPosts, post)
			}
		} else {
			// If we dont find the post in our map then we add it ti the posts to be checked for the trigger word and add it our map
			changedPosts = append(changedPosts, post)
			(*postAndNumberOfCommentsMap)[post.FullID] = post.NumberOfComments
		}
	}
	// Remove the keys we didn't access, we assume the post has turned old since it was NOT returned by client.Subreddit.NewPosts()
	for key := range *postAndNumberOfCommentsMap {
		if _, ok := accessedKeys[key]; !ok {
			delete(*postAndNumberOfCommentsMap, key)
		}
	}
	return changedPosts
}

// Finds and returns the all comments to a post
func FindPostComments(post *reddit.Post, postService *reddit.PostService, channel chan *reddit.PostAndComments, wait *sync.WaitGroup) {
	defer wait.Done()
	anon := func() (interface{}, error) {
		postAndComments, _, err := postService.Get(context.Background(), post.ID)
		return postAndComments, err
	}
	postAndComments := shared.RetryOnErrorWrapper(anon, "Error while getting post comments").(*reddit.PostAndComments)
	channel <- postAndComments
}

// Schedules go routines that find all the comments from various posts
func FindPostsCommentsScheduler(posts *[]*reddit.Post, postService *reddit.PostService) *[]*reddit.PostAndComments {
	channel := make(chan *reddit.PostAndComments)
	postsAndComments := make([]*reddit.PostAndComments, len(*posts))
	wait := &sync.WaitGroup{}
	mutex := sync.Mutex{}
	for _, post := range *posts {
		go FindPostComments(post, postService, channel, wait)
		mutex.Lock()
		wait.Add(1)
		mutex.Unlock()
	}
	go func() {
		defer close(channel)
		wait.Wait()
	}()
	index := 0
	for postAndComment := range channel {
		postsAndComments[index] = postAndComment
		index++
	}
	return &postsAndComments
}

func checkIfBotReplied(botUsername string, replies *[]*reddit.Comment) bool {
	for _, reply := range *replies {
		if reply.Author == botUsername {
			return true
		}
	}
	return false
}

// Recursively checks if the trigger was called on a comment or on its replies
func triggerCheck(botUsername string, triggerWords *[]string, comment *reddit.Comment, channel chan *map[string]string, wait *sync.WaitGroup, mutex *sync.Mutex) {
	defer wait.Done()
	queriedComment := make(map[string]string)
	commentBodyLowerCase := strings.ToLower(comment.Body)
	for _, triggerWord := range *triggerWords {
		idx := strings.Index(commentBodyLowerCase, triggerWord)
		if idx != -1 {
			// Remove the leading or trailing whitespaces that come after the trigger word then return the question e.g
			// "!sijui How to eat cake " to "How to eat cake"
			question := strings.TrimSpace(comment.Body[idx+len(triggerWord):])
			if len(question) > 0 {
				// First check if our bot has replied to the comment
				if !checkIfBotReplied(botUsername, &(comment.Replies.Comments)) {
					queriedComment[comment.FullID] = question
					channel <- &queriedComment
				}
			}
			break
		}

	}
	for index := range comment.Replies.Comments {
		mutex.Lock()
		wait.Add(1)
		mutex.Unlock()
		triggerCheck(botUsername, triggerWords, comment.Replies.Comments[index], channel, wait, mutex)
	}
}

// Check for the trigger word in the comments of a post
func CheckTriggerWord(botUsername string, triggerWords *[]string, postAndComments *reddit.PostAndComments, channel chan *map[string]string, wait *sync.WaitGroup, mutex *sync.Mutex) {
	defer wait.Done()
	// Convert the comment body to lower case then compare the result to our trigger words
	for _, comment := range postAndComments.Comments {
		mutex.Lock()
		wait.Add(1)
		mutex.Unlock()
		go triggerCheck(botUsername, triggerWords, comment, channel, wait, mutex)

	}
}

// Schedules go routines to check for the trigger word in the comments to posts(many)
func CheckTriggerWordScheduler(botUsername string, triggerWords *[]string, postsAndComments *[]*reddit.PostAndComments) *map[string]string {
	wait := sync.WaitGroup{}
	mutex := sync.Mutex{}
	channel := make(chan *map[string]string, len(*postsAndComments))
	queriedComments := make(map[string]string, 10)
	for _, postAndComments := range *postsAndComments {
		go CheckTriggerWord(botUsername, triggerWords, postAndComments, channel, &wait, &mutex)
		mutex.Lock()
		wait.Add(1)
		mutex.Unlock()
	}
	go func() {
		defer close(channel)
		wait.Wait()

	}()
	for queriedComment := range channel {
		for key, value := range *queriedComment {
			queriedComments[key] = value
		}
	}
	return &queriedComments
}

func Reply(commentID string, reply string, commentService *reddit.CommentService) {
	anon := func() (interface{}, error) {
		garbage, _, err := commentService.Submit(context.Background(), commentID, reply)
		return garbage, err
	}
	shared.RetryOnErrorWrapper(anon, "Error replying to comment")
}

func FetchNewPosts(client *reddit.Client, subreddit string) (*[]*reddit.Post, *reddit.Response, error) {
	posts, resp, err := client.Subreddit.NewPosts(context.Background(), subreddit, &reddit.ListOptions{Limit: 100})
	return &posts, resp, err
}

func FetchTopPosts(client *reddit.Client, subreddit string) (*[]*reddit.Post, *reddit.Response, error) {
	posts, resp, err := client.Subreddit.TopPosts(context.Background(), subreddit, &reddit.ListPostOptions{
		ListOptions: reddit.ListOptions{Limit: 100},
		Time:        "day"})
	return &posts, resp, err
}
