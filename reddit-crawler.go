package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"sync"
	"strings"
	"github.com/vartanbeno/go-reddit/v2/reddit"
)
var (credentials_path = "./credentials.json"
	subreddit = "kenya"
	post_options = reddit.ListPostOptions{ ListOptions: reddit.ListOptions{Limit: 100}, Time:"day"}
	trigger_words = []string{"!sijui-bot", "sijui-bot", "!sijui"}
	)

type CommentIDAndQuestion struct{
	comment_ID string
	question string
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
func CheckTriggerWord(trigger_words *[]string, post_and_comments *reddit.PostAndComments, channel chan *CommentIDAndQuestion, wait *sync.WaitGroup){
	defer wait.Done()
	comment_id_and_question := CommentIDAndQuestion{comment_ID: "", question: ""}
 	//Convert the comment body to lower case then compare the result to our trigger words
	for _, comment := range post_and_comments.Comments{
		comment_body_lower_case := strings.ToLower(comment.Body)
		for _, trigger_word := range *trigger_words{
			idx := strings.Index(comment_body_lower_case, trigger_word)
			if idx!=-1{
				//Remove the leading or trailing whitespaces that come after the trigger word then return the question e.g
				//"!sijui  how to eat cake  " -> "how to eat cake"
				question := strings.TrimSpace(comment.Body[idx+len(trigger_word):])
				if len(question) > 0{
					comment_id_and_question.comment_ID = comment.FullID
					comment_id_and_question.question = question
					log.Println(comment_id_and_question.comment_ID)
					log.Println("Triggered")
				}
				break
			}
		}
			
		}
	channel <- &comment_id_and_question
	}
	 

//Schedules go routines to check for the trigger word in the comments to posts(many)
func CheckTriggerWordScheduler(trigger_words *[]string, posts_and_comments *[]*reddit.PostAndComments)*[]*CommentIDAndQuestion{
	wait := sync.WaitGroup{}
	channel := make(chan *CommentIDAndQuestion, len(*posts_and_comments))
	queried_comments := make([]*CommentIDAndQuestion, 0, 10)
	log.Println("Checking for trigger word in comments to the posts")
	for _, post_and_comments := range *posts_and_comments{
		go CheckTriggerWord(trigger_words, post_and_comments, channel, &wait)
		wait.Add(1)
	}
	wait.Wait()
	close(channel)
	for queried_comment := range channel{
		//If we we actually found a trigger comment
		if queried_comment.comment_ID != ""{
			queried_comments = append(queried_comments, queried_comment)
		}
	}
	return &queried_comments
}
func TestReply(queried_comments *[]*(CommentIDAndQuestion), comment_sevice *reddit.CommentService){
	reply := "Ohio master"
	for _, queried_comment := range *queried_comments{
		log.Println("Replyiing .. .")
		comment_sevice.Submit(context.Background(), queried_comment.comment_ID, reply)
		log.Println(queried_comment.question)
		}
}
func main(){
	var credentials reddit.Credentials
	SetCredentials(&credentials)
	client := SetUpClient(&credentials)
	posts, _, err := client.Subreddit.TopPosts(context.Background(), subreddit, &post_options)
	post_service := client.Post
	//comment_service := client.Comment
	if err != nil{
		log.Fatal("Error while getting top posts ", err)
	}
	posts_and_comments := FindPostsCommentsScheduler(&posts, post_service)
	queried_comments := CheckTriggerWordScheduler(&trigger_words, posts_and_comments)
	log.Println(len(*queried_comments))
	//TestReply(queried_comments, comment_service)
	
}