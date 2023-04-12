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
var (credentials_path = "./credentials.json"
	post_and_number_of_comments_JSON_path = "./posts_comment_count.json"
	subreddit = "kenya"
	//post_options = reddit.ListPostOptions{ ListOptions: reddit.ListOptions{Limit: 100}, Time:"day"}
	post_options = reddit.ListOptions{Limit: 100}
	trigger_words = []string{"!sijui-bot", "sijui-bot", "!sijui"}
	post_and_number_of_comments_map = make(map[string]int)
	)

type CommentIDAndQuestion struct{
	comment_ID string
	question string
}

//Reads the json file containing the bots credentials for authentification in order to access the Reddit API
func SetCredentials(credentials *reddit.Credentials){
	file, err := os.Open(credentials_path)
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
func CreatePostsNumberOfCommentsJSON(post_and_number_of_comments_map *map[string]int, post_and_number_of_comments_JSON_path *string, posts *[]*reddit.Post){
	//Create a new map and fill it with post IDs and the number of comments
	for _, post := range *posts{
		(*post_and_number_of_comments_map)[post.FullID] = post.NumberOfComments
	}	
	//Create a JSON file to store the map
	file, err := os.Create(*post_and_number_of_comments_JSON_path)
	if err != nil{log.Fatal("Error creating the posts_and_comment_count.json file", err)}
	defer file.Close()	
	//Store the map to the created JSON file
	encoder := json.NewEncoder(file)
	if err := encoder.Encode(post_and_number_of_comments_map); err != nil{ log.Fatal("Error storing values to the JSON ", err)}
}

func CheckIfPostsHaveNewComments(post_and_number_of_comments_map *map[string]int, posts *[]*reddit.Post) []*reddit.Post{
	changed_posts := make([]*reddit.Post, 0, len(*posts))
	accessed_keys := make(map[string]bool)
	for _, post := range *posts{
		//If we have the post in our map
		if _, ok := (*post_and_number_of_comments_map)[post.FullID]; ok{
			//If the number of comments on the post have increased
			if post.NumberOfComments > (*post_and_number_of_comments_map)[post.FullID]{
				(*post_and_number_of_comments_map)[post.FullID] = post.NumberOfComments
				//add the post to the posts to be checked for the trigger word
				changed_posts = append(changed_posts, post)
				//if the number of comments on the post has reduced to handle edge cases where a user deletes a comment
				}else if post.NumberOfComments < (*post_and_number_of_comments_map)[post.FullID]{
					//Change to the new reduced number of comments
					(*post_and_number_of_comments_map)[post.FullID] = post.NumberOfComments
				}
				//update the accessed keys
				accessed_keys[post.FullID] = true
		}else{
			//If we dont find the post in our map then we add it ot the posts to be checked for the trigger word and add it to the map
			changed_posts = append(changed_posts, post)
			(*post_and_number_of_comments_map)[post.FullID] = post.NumberOfComments
		}
	}
	//Remove the keys we didn't access, we assume the post has turned old since it was returned by client.Subreddit.NewPosts()
	for _, post := range *posts{
		if _, ok := accessed_keys[post.FullID]; !ok{
			delete(*post_and_number_of_comments_map, post.FullID)
		}
	}
	return changed_posts
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
	//Convert the comment body to lower case then compare the result to our trigger words
	for _, comment := range post_and_comments.Comments{
		queried_comment := CommentIDAndQuestion{comment_ID: "", question: ""}
		comment_body_lower_case := strings.ToLower(comment.Body)
		for _, trigger_word := range *trigger_words{
			idx := strings.Index(comment_body_lower_case, trigger_word)
			if idx!=-1{
				//Remove the leading or trailing whitespaces that come after the trigger word then return the question e.g
				//"!sijui  how to eat cake  " -> "how to eat cake"
				question := strings.TrimSpace(comment.Body[idx+len(trigger_word):])
				if len(question) > 0{
					queried_comment.comment_ID = comment.FullID
					queried_comment.question = question
					log.Println("Triggered")
					channel <- &queried_comment
				}
				break
			}
		}
			
		}
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
		queried_comments = append(queried_comments, queried_comment)
	}
	return &queried_comments
}
func TestReply(queried_comments *[]*(CommentIDAndQuestion), comment_sevice *reddit.CommentService){
	reply := "Konichiwa master"
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
	posts, _, err:= client.Subreddit.NewPosts(context.Background(), subreddit, &post_options)
	if err != nil{log.Fatal("Error while getting posts ", err)}
	log.Printf("ID %v", posts[0].FullID)
	log.Printf("Number Comments %v", posts[0].NumberOfComments)
	if !CheckIfPostsNumberOfCommentsJSONExists(&post_and_number_of_comments_JSON_path){
		CreatePostsNumberOfCommentsJSON(&post_and_number_of_comments_map, &post_and_number_of_comments_JSON_path, &posts)
	}
	posts = CheckIfPostsHaveNewComments(&post_and_number_of_comments_map, &posts)
	CreatePostsNumberOfCommentsJSON(&post_and_number_of_comments_map, &post_and_number_of_comments_JSON_path, &posts)
	// posts, _, err := client.Subreddit.TopPosts(context.Background(), subreddit, &post_options)
	post_service := client.Post
	comment_service := client.Comment
	posts_and_comments := FindPostsCommentsScheduler(&posts, post_service)
	queried_comments := CheckTriggerWordScheduler(&trigger_words, posts_and_comments)
	log.Println(len(*queried_comments))
	TestReply(queried_comments, comment_service)
	
}