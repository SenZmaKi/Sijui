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
	repliedCommentsPath = "./replied_comments.json"
	subreddit = "kenya"
	//postOptions = reddit.ListPostOptions{ ListOptions: reddit.ListOptions{Limit: 100}, Time:"day"}
	postOptions = reddit.ListOptions{Limit: 100}
	triggerWords = []string{"!sijui-bot", "sijui-bot", "!sijui"}
	postAndNumberOfCommentsMap = make(map[string]int)
	repliedCommentsMap = make(map[string]bool)
	)

// type CommentIDAndQuestion struct{
// 	commentID string
// 	question string
// }

//Reads the json file containing the bots credentials for authentification in order to access the Reddit API
func SetCredentials(credentials *reddit.Credentials, credentialsPath string){
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
func CheckIfPostsNumberOfCommentsJSONExists(postAndNumberOfCommentsJsonPath *string)bool{
	if _, err := os.Stat(*postAndNumberOfCommentsJsonPath); err == nil{
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
	if err := encoder.Encode(postAndNumberOfCommentsMap); err != nil{ log.Println("Error writing values read from map to the JSON ", err)}
}

func UpdateJSONWithPostsNumberOfCommentsMap(postAndNumberOfCommentsMap *map[string]int, postAndNumberOfCommentsJsonPath *string){
	//Using Create instead of OpenFIle might result to undefinded behaviour for cases where you want to specifically open the file for writing and not create it if it doesn't exist
	if file, err := os.Create(*postAndNumberOfCommentsJsonPath); err!=nil{
		log.Println("Error opening JSON in attempt to update JSON with the new(changed) post and number of comments read from map ", err)
	}else{
		defer file.Close()
		encoder := json.NewEncoder(file)
		if err := encoder.Encode(postAndNumberOfCommentsMap); err != nil{log.Println("Error updating JSON with the new post and number of comments read from map ", err)}
	}
	
}

func WriteJsonToPostsNumberOFCommentsMap(postsNumberOfCommentsMap *map[string]int, postAndNumberOfCommentsJsonPath *string){
		if file, err := os.Open(*postAndNumberOfCommentsJsonPath); err!=nil{
			log.Println("Error while opening the JSON file in an attempt to write JSON to map ", err)
		}else{
			defer file.Close()
			decoder := json.NewDecoder(file)
			if err := decoder.Decode(postsNumberOfCommentsMap); err != nil{log.Println("Error writing values read from from JSON to map ", err)}
		}
		
	}


func FindPostsThatHaveHaveNewComments(postAndNumberOfCommentsMap *map[string]int, posts *[]*reddit.Post) []*reddit.Post{
	changedPosts := make([]*reddit.Post, 0, len(*posts)/4)
	accessedKeys := make(map[string]bool)
	for _, post := range *posts{
		//update the accessed keys
		accessedKeys[post.FullID] = true
		//If we have the post in our map
		if _, ok := (*postAndNumberOfCommentsMap)[post.FullID]; ok{
			//If the number of comments on the post have increased
			if post.NumberOfComments > (*postAndNumberOfCommentsMap)[post.FullID]{
				(*postAndNumberOfCommentsMap)[post.FullID] = post.NumberOfComments
				//add the post to the posts to be checked for the trigger word
				changedPosts = append(changedPosts, post)
				//if the number of comments on the post has reduced to handle edge cases where a user deletes a comment
				}else if post.NumberOfComments < (*postAndNumberOfCommentsMap)[post.FullID]{
					//Change to the new reduced number of comments
					(*postAndNumberOfCommentsMap)[post.FullID] = post.NumberOfComments
					//add the post to the posts to be checked for the trigger word
					changedPosts = append(changedPosts, post)
				}
		}else{
			//If we dont find the post in our map then we add it ti the posts to be checked for the trigger word and add it our map
			println(post.Title, " ", post.NumberOfComments)
			changedPosts = append(changedPosts, post)
			(*postAndNumberOfCommentsMap)[post.FullID] = post.NumberOfComments
		}
	}
	//Remove the keys we didn't access, we assume the post has turned old since it was NOT returned by client.Subreddit.NewPosts()
	for key := range *postAndNumberOfCommentsMap{
		if _, ok := accessedKeys[key]; !ok{
			println("Deleted")
			delete(*postAndNumberOfCommentsMap, key)
		}
	}
	return changedPosts
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
func trigger_check(triggerWords*[]string, comment *reddit.Comment, channel chan *map[string]string, wait *sync.WaitGroup, mutex *sync.Mutex){
	defer wait.Done()
	queriedComment := make(map[string]string)
	commentBodyLowerCase := strings.ToLower(comment.Body)
	for _, trigger_word := range *triggerWords{
		idx := strings.Index(commentBodyLowerCase, trigger_word)
		if idx!=-1{
			//Remove the leading or trailing whitespaces that come after the trigger word then return the question e.g
			//"!sijui  how to eat cake  " -> "how to eat cake"
			question := strings.TrimSpace(comment.Body[idx+len(trigger_word):])
			if len(question) > 0{
				queriedComment[comment.FullID] = question
				log.Println("Triggered")
				channel <- &queriedComment
			}
			break
		}
	}
		for index := range comment.Replies.Comments{
			mutex.Lock()
			wait.Add(1)
			mutex.Unlock()
			trigger_check(triggerWords, comment.Replies.Comments[index], channel, wait, mutex)
		}
	}


//Check for the trigger word in the comments of a post
func CheckTriggerWord(triggerWords *[]string, postAndComments *reddit.PostAndComments, channel chan *map[string]string, wait *sync.WaitGroup, mutex *sync.Mutex){
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
func CheckTriggerWordScheduler(triggerWords *[]string, postsAndComments *[]*reddit.PostAndComments)*map[string]string{
	wait := sync.WaitGroup{}
	mutex := sync.Mutex{}
	channel := make(chan *map[string]string, len(*postsAndComments))
	queriedComments := make(map[string]string, 10)
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
	for queriedComment := range channel{
		for key, value := range *queriedComment{
			queriedComments[key] = value
		}
	}
	return &queriedComments
}
// func CheckIfRepliedCommentsJSONExists(repliedCommentsPath *string) bool{
// 	if _, err:= os.Stat(*repliedCommentsPath); err!=nil{return false}
// 	return true
// }
func UpdateRepliedCommentsJSON(repliedCommentsPath *string, queriedComments *map[string]string){
	if file, err := os.Create(*repliedCommentsPath); err!=nil{
		log.Println("Error opening replied comments JSON file ", err)
	}else{
		defer file.Close()
		encoder := json.NewEncoder(file)
		if err := encoder.Encode(queriedComments); err!=nil{log.Println("Error writing comment to repliedComments JSON file ", err)}
	}

}

func LookForUnrepliedComments(repliedCommentsPath *string, queriedComments *map[string]string)*map[string]string{
	repliedCommentsMap := make(map[string]string)
	unrepliedComments := make(map[string]string)
	accessedKeys := make(map[string]bool)
	if file, err := os.Create(*repliedCommentsPath); err!=nil{
		log.Println("Error opening replied comments JSON file ", err)
	}else{
		defer file.Close()
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&repliedCommentsMap); err!=nil{log.Println("Error reading comment from repliedCommentsJSON to write to repliedCommentsMap ", err)}
		for key, value := range *queriedComments{
			if _, ok := repliedCommentsMap[key]; !ok{
				unrepliedComments[key] = value
				accessedKeys[key] = true
			}else{
				accessedKeys[key] = true
			}
		}
		for key := range *queriedComments{
			if _, ok := accessedKeys[key]; !ok{
				delete(repliedCommentsMap, key)
			}
		}
	}
	return &unrepliedComments
	
}

func TestReply(unrepliedComments *map[string]string, comment_sevice *reddit.CommentService){
	reply := "Konichiwa master"
	for commentID, question := range *unrepliedComments{
		log.Println("Replyiing .. .")
		comment_sevice.Submit(context.Background(), commentID, reply)
		log.Println(question)
		}
}
func main(){
	var credentials reddit.Credentials
	SetCredentials(&credentials, credentialsPath)
	client := SetUpClient(&credentials)
	posts, _, err:= client.Subreddit.NewPosts(context.Background(), subreddit, &postOptions)
	if err != nil{log.Fatal("Error while getting posts ", err)}
	//log.Printf("ID %v", posts[0].FullID)
	//log.Printf("Number Comments %v", posts[0].NumberOfComments)
	if !CheckIfPostsNumberOfCommentsJSONExists(&postAndNumberOfCommentsJsonPath){
		CreatePostsNumberOfCommentsJSON(&postAndNumberOfCommentsMap, &postAndNumberOfCommentsJsonPath, &posts)
	}
	//Read from the stored json and map the values to the map
	WriteJsonToPostsNumberOFCommentsMap(&postAndNumberOfCommentsMap, &postAndNumberOfCommentsJsonPath)
	posts = FindPostsThatHaveHaveNewComments(&postAndNumberOfCommentsMap, &posts)
	//Update the json with the changed posts
	UpdateJSONWithPostsNumberOfCommentsMap(&postAndNumberOfCommentsMap, &postAndNumberOfCommentsJsonPath)
	//CreatePostsNumberOfCommentsJSON(&postAndNumberOfCommentsMap, &postAndNumberOfCommentsJsonPath, &posts)
	//posts, _, err := client.Subreddit.TopPosts(context.Background(), subreddit, &postOptions)
	postService := client.Post
	//comment_service := client.Comment
	postsAndComments := FindPostsCommentsScheduler(&posts, postService)
	queriedComments := CheckTriggerWordScheduler(&triggerWords, postsAndComments)
	for commentID, question := range *queriedComments{
		log.Println(commentID, ": ", question)
	}
	//TestReply(queriedComments, comment_service)
	
}