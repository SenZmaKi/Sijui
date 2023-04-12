package main

import (
	"context"
	"encoding/json"
	"os"
	"log"
	"sync"
	"strings"
	"github.com/vartanbeno/go-reddit/v2/reddit"
)
var (credentialsPath = "./credentials.json"
	postAndNumberOfCommentsJsonPath = "./posts_comment_count.json"
	subreddit = "kenya"
	//postOptions = reddit.ListPostOptions{ ListOptions: reddit.ListOptions{Limit: 100}, Time:"day"}
	postOptions = reddit.ListOptions{Limit: 100}
	triggerWords = []string{"!sijui-bot", "sijui-bot", "!sijui"}
	postAndNumberOfCommentsMap = make(map[string]int)
	)

type CommentIDAndQuestion struct{
	commentID string
	question string
}

//Reads the json file containing the bots credentials for authentification in order to access the Reddit API
func SetCredentials(credentials *reddit.Credentials){
	file, err := os.Open(credentialsPath)
	if err != nil{
		log.Fatal("Error while reading credentials", err)
	}
	defer file.Close()
	err = json.NewDecoder(file).Decode(credentials)
	if err != nil{
		log.Fatal("Error during Unmarshal() ", err)
	}
	//log.Printf("Username: %v, Password: %v,Id: %v, Secret: %v", credentials.Username, credentials.Password, credentials.ID, credentials.Secret)

}

//Sets up reddit API client using the provided credentials
func SetUpClient(credentials *reddit.Credentials) *reddit.Client{
	client, err := reddit.NewClient(*credentials)
	if err != nil{
		log.Fatal("Error while setting up client ", err)
	}
	return client

}
func CheckIfPostsNumberOfCommentsJSONExists(posts_comment_count_path *string)bool{
	if _, err := os.Stat(*posts_comment_count_path); err == nil{
		return true
	}else{return false}
}
func CreatePostsNumberOfCommentsJSON(postAndNumberOfCommentsMap *map[string]int, postAndNumberOfCommentsJsonPath *string, posts *[]*reddit.Post){
	//Create a new map and fill it with post IDs and the number of comments
	for _, post := range *posts{
		(*postAndNumberOfCommentsMap)[post.FullID] = post.NumberOfComments
	}	
	//Create a JSON file to store the map
	file, err := os.Create(*postAndNumberOfCommentsJsonPath)
	if err != nil{log.Fatal("Error creating the posts_and_comment_count.json file", err)}
	defer file.Close()	
	//Store the map to the created JSON file
	encoder := json.NewEncoder(file)
	if err := encoder.Encode(postAndNumberOfCommentsMap); err != nil{ log.Fatal("Error storing values to the JSON ", err)}
}

func CheckIfPostsHaveNewComments(postAndNumberOfCommentsMap *map[string]int, posts *[]*reddit.Post) []*reddit.Post{
	changed_posts := make([]*reddit.Post, 0, len(*posts))
	accessed_keys := make(map[string]bool)
	for _, post := range *posts{
		//If we have the post in our map
		if _, ok := (*postAndNumberOfCommentsMap)[post.FullID]; ok{
			//If the number of comments on the post have increased
			if post.NumberOfComments > (*postAndNumberOfCommentsMap)[post.FullID]{
				(*postAndNumberOfCommentsMap)[post.FullID] = post.NumberOfComments
				//add the post to the posts to be checked for the trigger word
				changed_posts = append(changed_posts, post)
				//if the number of comments on the post has reduced to handle edge cases where a user deletes a comment
				}else if post.NumberOfComments < (*postAndNumberOfCommentsMap)[post.FullID]{
					//Change to the new reduced number of comments
					(*postAndNumberOfCommentsMap)[post.FullID] = post.NumberOfComments
				}
				//update the accessed keys
				accessed_keys[post.FullID] = true
		}else{
			//If we dont find the post in our map then we add it ot the posts to be checked for the trigger word and add it to the map
			changed_posts = append(changed_posts, post)
			(*postAndNumberOfCommentsMap)[post.FullID] = post.NumberOfComments
		}
	}
	//Remove the keys we didn't access, we assume the post has turned old since it was returned by client.Subreddit.NewPosts()
	for _, post := range *posts{
		if _, ok := accessed_keys[post.FullID]; !ok{
			delete(*postAndNumberOfCommentsMap, post.FullID)
		}
	}
	return changed_posts
}

//Finds and returns the all comments to a post
func FindPostComments(post *reddit.Post, postService *reddit.PostService, channel chan *reddit.PostAndComments, wait *sync.WaitGroup){
	defer wait.Done()
	postAndComments, _, err := postService.Get(context.Background(), post.ID)
	if err != nil{
		log.Fatal("Error while getting post comments ", err)
	}
	channel <- postAndComments
}

//Schedules go routines that find all the comments from various posts
func FindPostsCommentsScheduler(posts *[]*reddit.Post, postService *reddit.PostService)*[]*reddit.PostAndComments{
	channel := make(chan *reddit.PostAndComments)
	postsAndComments := make([]*reddit.PostAndComments, len(*posts))
	wait := &sync.WaitGroup{}
	mutex := sync.Mutex{}
	log.Printf("Finding comments to %v posts.. .", len(*posts))
	for _, post := range *posts{
		go FindPostComments(post, postService, channel, wait)
		mutex.Lock()
		wait.Add(1)
		mutex.Unlock()
	}
	go func(){
		defer close(channel)
		wait.Wait()
	}()
	index := 0
	for postAndComment := range channel{
		postsAndComments[index] = postAndComment
		index++
	}
	return &postsAndComments
}

//Recursively checks if the trigger was called on a comment or on it's replies
func trigger_check(triggerWords*[]string, comment *reddit.Comment, channel chan *CommentIDAndQuestion, wait *sync.WaitGroup, mutex *sync.Mutex){
	defer wait.Done()
	queried_comment := CommentIDAndQuestion{commentID: "", question: ""}
	commentBodyLowerCase := strings.ToLower(comment.Body)
	for _, trigger_word := range *triggerWords{
		idx := strings.Index(commentBodyLowerCase, trigger_word)
		if idx!=-1{
			//Remove the leading or trailing whitespaces that come after the trigger word then return the question e.g
			//"!sijui  how to eat cake  " -> "how to eat cake"
			question := strings.TrimSpace(comment.Body[idx+len(trigger_word):])
			if len(question) > 0{
				queried_comment.commentID = comment.FullID
				queried_comment.question = question
				log.Println("Triggered")
				channel <- &queried_comment
			}
			break
		}
	}
	if len(comment.Replies.Comments) > 0{
		for index := range comment.Replies.Comments{
			mutex.Lock()
			wait.Add(1)
			mutex.Unlock()
			trigger_check(triggerWords, comment.Replies.Comments[index], channel, wait, mutex)
		}
	}
}

//Check for the trigger word in the comments of a post
func CheckTriggerWord(triggerWords *[]string, postAndComments *reddit.PostAndComments, channel chan *CommentIDAndQuestion, wait *sync.WaitGroup, mutex *sync.Mutex){
	defer wait.Done()
	//Convert the comment body to lower case then compare the result to our trigger words
	for _, comment := range postAndComments.Comments{
		mutex.Lock()
		wait.Add(1)
		mutex.Unlock()
		go trigger_check(triggerWords, comment, channel, wait, mutex)

		}
	}
	 

//Schedules go routines to check for the trigger word in the comments to posts(many)
func CheckTriggerWordScheduler(triggerWords *[]string, postsAndComments *[]*reddit.PostAndComments)*[]*CommentIDAndQuestion{
	wait := sync.WaitGroup{}
	mutex := sync.Mutex{}
	channel := make(chan *CommentIDAndQuestion, len(*postsAndComments))
	queried_comments := make([]*CommentIDAndQuestion, 0, 10)
	log.Println("Checking for trigger word in comments to the posts")
	for _, postAndComments := range *postsAndComments{
		go CheckTriggerWord(triggerWords, postAndComments, channel, &wait, &mutex)
		mutex.Lock()
		wait.Add(1)
		mutex.Unlock()
	}
	go func(){
		defer close(channel)
		wait.Wait()

	}()
	for queried_comment := range channel{
		queried_comments = append(queried_comments, queried_comment)
	}
	return &queried_comments
}
func TestReply(queried_comments *[]*(CommentIDAndQuestion), comment_sevice *reddit.CommentService){
	reply := "Konichiwa master"
	for _, queried_comment := range *queried_comments{
		log.Println("Replyiing .. .")
		comment_sevice.Submit(context.Background(), queried_comment.commentID, reply)
		log.Println(queried_comment.question)
		}
}
func main(){
	var credentials reddit.Credentials
	SetCredentials(&credentials)
	client := SetUpClient(&credentials)
	posts, _, err:= client.Subreddit.NewPosts(context.Background(), subreddit, &postOptions)
	if err != nil{log.Fatal("Error while getting posts ", err)}
	log.Printf("ID %v", posts[0].FullID)
	log.Printf("Number Comments %v", posts[0].NumberOfComments)
	if !CheckIfPostsNumberOfCommentsJSONExists(&postAndNumberOfCommentsJsonPath){
		CreatePostsNumberOfCommentsJSON(&postAndNumberOfCommentsMap, &postAndNumberOfCommentsJsonPath, &posts)
	}
	posts = CheckIfPostsHaveNewComments(&postAndNumberOfCommentsMap, &posts)
	CreatePostsNumberOfCommentsJSON(&postAndNumberOfCommentsMap, &postAndNumberOfCommentsJsonPath, &posts)
	//posts, _, err := client.Subreddit.TopPosts(context.Background(), subreddit, &postOptions)
	postService := client.Post
	//comment_service := client.Comment
	postsAndComments := FindPostsCommentsScheduler(&posts, postService)
	queried_comments := CheckTriggerWordScheduler(&triggerWords, postsAndComments)
	for _, queried_comment := range *queried_comments{
		log.Println(queried_comment.question)
	}
	//TestReply(queried_comments, comment_service)
	
}