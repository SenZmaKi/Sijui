package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"sync"
	"github.com/vartanbeno/go-reddit/v2/reddit"
)
var (credentials_path = "./credentials.json"
	subreddit = "kenya"
	post_options = reddit.ListPostOptions{ ListOptions: reddit.ListOptions{Limit: 100}, Time:"day"}
	trigger_word = "!Sijui"
	)

type PostTriggerComments struct{
	comment_indices []int
	post_and_comments *reddit.PostAndComments
}

//Reads the json file containing the bots credentials for authentification in order to access the Reddit API
func SetCredentials(credentials *reddit.Credentials){
	content, err := ioutil.ReadFile(credentials_path)
	if err != nil{
		log.Fatal("Error while reading credentials", err)
	}
	err = json.Unmarshal(content, credentials)
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
//Finds and returns the all comments to a post
func FindPostComments(post *reddit.Post, post_service *reddit.PostService, channel chan *reddit.PostAndComments, wait *sync.WaitGroup){
	defer wait.Done()
	post_and_comments, _, err := post_service.Get(context.Background(), post.ID)
	if err != nil{
		log.Fatal("Error while getting post comments ", err)
	}
	channel <- post_and_comments
}

//Schedules go routines that find all the comments from various posts
func FindPostsCommentsScheduler(posts *[]*reddit.Post, post_service *reddit.PostService)*[]*reddit.PostAndComments{
	channel := make(chan *reddit.PostAndComments, len(*posts))
	posts_and_comments := make([]*reddit.PostAndComments, len(*posts))
	wait := &sync.WaitGroup{}
	log.Printf("Finding comments to %v posts.. .", len(*posts))
	for _, post := range *posts{
		go FindPostComments(post, post_service, channel, wait)
		wait.Add(1)
	}
	wait.Wait()
	close(channel)
	index := 0
	for post_and_comment := range channel{
		posts_and_comments[index] = post_and_comment
		index++
	}
	return &posts_and_comments
}

//Check for the trigger word in the comments of a post
func CheckTriggerWord(trigger_word *string, post_and_comments *reddit.PostAndComments, channel chan *PostTriggerComments, wait *sync.WaitGroup){
	defer wait.Done()
	post_trigger_comments := PostTriggerComments{comment_indices: []int{}, post_and_comments: post_and_comments }
	for index, comment := range post_and_comments.Comments{
		if comment.Body == *trigger_word{
			log.Println("Triggered")
			post_trigger_comments.comment_indices = append(post_trigger_comments.comment_indices, index)
		}
	}
	channel <- &post_trigger_comments
	}
	 

//Schedules go routines to check for the trigger word in the comments to posts(many)
func CheckTriggerWordScheduler(trigger_word *string, posts_and_comments *[]*reddit.PostAndComments)*[]*PostTriggerComments{
	wait := sync.WaitGroup{}
	channel := make(chan *PostTriggerComments, len(*posts_and_comments))
	posts_trigger_comments := make([]*PostTriggerComments, 0, 10)
	log.Println("Checking for trigger word in comments to the posts")
	for _, post_and_comments := range *posts_and_comments{
		go CheckTriggerWord(trigger_word, post_and_comments, channel, &wait)
		wait.Add(1)
	}
	wait.Wait()
	close(channel)
	for post_trigger_comments := range channel{
		posts_trigger_comments = append(posts_trigger_comments, post_trigger_comments)
	}
	return &posts_trigger_comments
}

func main(){
	var credentials reddit.Credentials
	SetCredentials(&credentials)
	client := SetUpClient(&credentials)
	posts, _, err := client.Subreddit.TopPosts(context.Background(), subreddit, &post_options)
	post_service := client.Post
	if err != nil{
		log.Fatal("Error while getting top posts ", err)
	}
	posts_and_comments := FindPostsCommentsScheduler(&posts, post_service)
	posts_trigger_comments := CheckTriggerWordScheduler(&trigger_word, posts_and_comments)
	for _, post_trigger_comments := range *posts_trigger_comments{
		for _, trigger_comments := range post_trigger_comments.comment_indices{
			log.Println(post_trigger_comments.post_and_comments.Comments[trigger_comments].Body)
		}
	}
	
}