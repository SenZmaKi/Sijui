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

type GoogleResult struct {
	Title   string
	Snippet string
	Link    string
}

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

func SetUpGoogleSearchService(credentials *map[string]string) *customsearch.CseListCall {
	anon := func() (interface{}, error) {
		return customsearch.NewService(context.Background(), option.WithAPIKey((*credentials)["CustomSearchAPIKey"]))
	}
	service := shared.RetryOnErrorWrapper(anon, "Error creating new custom search service").(*customsearch.Service)
	return service.Cse.List().Cx((*credentials)["SearchEngineID"])
}

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

func SetUpOpenAIClient(credentials *map[string]string) *openai.Client {
	return openai.NewClient((*credentials)["OpenAIAPIKey"])
}

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
	println(response.Choices[0].Message.Content)
	if err != nil {
		return answer, err
	}
	return response.Choices[0].Message.Content, err
}
