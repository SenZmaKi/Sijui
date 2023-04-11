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
	)

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
func SetUpClient(credentials *reddit.Credentials) *reddit.Client{
	client, err := reddit.NewClient(*credentials)
	if err != nil{
		log.Fatal("Error while setting up client ", err)
	}
	return client

}
 func FindPostComments(post *reddit.Post, post_service *reddit.PostService, channel chan *reddit.PostAndComments, wait *sync.WaitGroup){
	 	defer wait.Done()
		post_and_comments, _, err := post_service.Get(context.Background(), post.ID)
		if err != nil{
			log.Fatal("Error while getting post comments ", err)
		}
		channel <- post_and_comments
 }

 func FindPostsCommentsRoutine(posts *[]*reddit.Post, post_service *reddit.PostService)*[]*reddit.PostAndComments{
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
		for post_and_comment := range channel{
			posts_and_comments = append(posts_and_comments, post_and_comment)
		}
		return &posts_and_comments
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
	posts_and_comments := FindPostsCommentsRoutine(&posts, post_service)
	
}