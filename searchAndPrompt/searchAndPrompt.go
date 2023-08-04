package searchAndPrompt

import (
	"context"
	"encoding/json"
	"os"

	"shared"

	"github.com/sashabaranov/go-openai"
	"google.golang.org/api/customsearch/v1"
	"google.golang.org/api/option"
)

// Provides a more structured format to the fetched google results
type GoogleResult struct {
	Title   string
	Snippet string
	Link    string
}

// Reads the file containing google credentials and sets up googleCredentials with the read credentials
func SetUpGoogleCredentials(credentialsPath string) *map[string]string {
	googleCredentials := make(map[string]string)
	if file, err := os.Open(credentialsPath); err != nil {
		shared.LogError("Error opening googleCredentials.json", err)
	} else {
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&googleCredentials); err != nil {
			shared.LogError("Error decoding googleCredentials.json into googleCredentials map", err)
		}
	}
	return &googleCredentials
}

// Sets up a custom google search service using the passed credentials, think of it as setting up the google search api
func SetUpGoogleSearchService(credentials *map[string]string) *customsearch.CseListCall {
	anon := func() (interface{}, error) {
		return customsearch.NewService(context.Background(), option.WithAPIKey((*credentials)["CustomSearchAPIKey"]))
	}
	service := shared.RetryOnErrorWrapper(anon, "Error creating new custom search service").(*customsearch.Service)
	return service.Cse.List().Cx((*credentials)["SearchEngineID"])
}

// Makes a google search using the passed query and returns the first 3 results as list containing 3 googleResults structs or an error
func GoogleSearch(query string, searchService *customsearch.CseListCall) (*[]GoogleResult, error) {
	results, err := searchService.Q(query).Do()
	googleResults := make([]GoogleResult, 3)
	if err == nil {
		resultItems := results.Items[:3]
		for idx := 0; idx < 3; idx++ {
			googleResults[idx].Title = resultItems[idx].Title
			googleResults[idx].Snippet = resultItems[idx].Snippet
			googleResults[idx].Link = resultItems[idx].Link
		}
	}
	return &googleResults, err
}

// Reads the OpenAI credentials and sets up openAICredentials with the read values
func SetUpOpenAICredentials(credentialsPath string) *map[string]string {
	openAICredentials := make(map[string]string)
	if file, err := os.Open(credentialsPath); err != nil {
		shared.LogError("Error opening openAICredentials.json", err)
	} else {
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&openAICredentials); err != nil {
			shared.LogError("Error decoding openAICredentials.json into openAICredentials map", err)
		}
	}
	return &openAICredentials
}

// Sets up an openAI client using the passed credentials
func SetUpOpenAIClient(credentials *map[string]string) *openai.Client {
	return openai.NewClient((*credentials)["OpenAIAPIKey"])
}

// Prompts ChatGpt (GPT 3.5)
func PromptGpt(client *openai.Client, prompt string) (string, error) {
	request := openai.ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	}
	answer := "nil"
	response, err := client.CreateChatCompletion(context.Background(), request)
	if err == nil {
		answer = response.Choices[0].Message.Content
	}
	return answer, err
}
