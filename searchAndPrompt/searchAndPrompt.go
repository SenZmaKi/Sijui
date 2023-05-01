package searchAndPrompt

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/sashabaranov/go-openai"
	"google.golang.org/api/customsearch/v1"
	"google.golang.org/api/option"
)

type GoogleResult struct {
	Title   string
	Snippet string
	Link    string
}

func SetUpGoogleCredentials(credentialsPath *string) *map[string]string {
	googleCredentials := make(map[string]string)
	if file, err := os.Open(*credentialsPath); err != nil {
		log.Fatal("Error opening googleCredentials.json ", err)
	} else {
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&googleCredentials); err != nil {
			log.Fatal("Error decoding googleCredentials.json into googleCredentials map ", err)
		}
	}
	return &googleCredentials
}

func SetUpGoogleSearchService(credentials *map[string]string) *customsearch.CseListCall {
	service, err := customsearch.NewService(context.Background(), option.WithAPIKey((*credentials)["CustomSearchAPIKey"]))
	if err != nil {
		log.Fatal("Error creating new custom search service ", err)
	}
	return service.Cse.List().Cx((*credentials)["SearchEngineID"])
}

func GoogleSearch(query *string, searchService *customsearch.CseListCall) *[]GoogleResult {
	results, err := searchService.Q(*query).Do()
	if err != nil {
		log.Println("Error fetching google search results ", err)
	}
	resultItems := results.Items[:3]
	googleResults := make([]GoogleResult, 3)
	for idx := 0; idx < 3; idx++ {
		googleResults[idx].Title = resultItems[idx].Title
		googleResults[idx].Snippet = resultItems[idx].Snippet
		googleResults[idx].Link = resultItems[idx].Link
	}
	return &googleResults
}

func SetUpOpenAICredentials(credentialsPath *string) *map[string]string {
	openAICredentials := make(map[string]string)
	if file, err := os.Open(*credentialsPath); err != nil {
		log.Fatal("Error opening openAICredentials.json")
	} else {
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&openAICredentials); err != nil {
			log.Fatal("Error decoding openAICredentials.json into openAICredentials map ", err)
		}
	}
	return &openAICredentials
}

func SetUpOpenAIClient(credentials *map[string]string) *openai.Client {
	return openai.NewClient((*credentials)["OpenAIAPIKey"])
}

func PromptGpt(client *openai.Client, prompt *string) *string {
	request := openai.ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: *prompt,
			},
		},
	}
	answer := "nil"
	response, err := client.CreateChatCompletion(context.Background(), request)
	if err != nil {
		log.Println("Error getting response from ChatGpt ", err)
		return &answer
	}
	return &response.Choices[0].Message.Content
}

//func main(){

// client := SetUpOpenAIClient(SetUpGoogleCredentials(&openAICredentialsPath))
// prompt := "Korewa jiyuu da"
// answer := PromptGpt(client, &prompt)
// print(*answer)

// googleCredentials := SetUpGoogleCredentials(&googleCredentialsPath)
// searchService := SetUpGoogleSearchService(googleCredentials)
// query := "How to shit"
// googleResults := GoogleSearch(&query, searchService)
// for _, googleResult := range *googleResults{
// 	println("Title: ", googleResult.Title)
// 	println("Snippet: ", googleResult.Snippet)
// 	println("Link: ", googleResult.Link)
// }
//}
