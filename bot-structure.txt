SIJUI

Reddit Crawler - crawls through a specifc subreddit in a specific order for a comment containing
                 the command keyword then returns said comment
    - Reddit API
    - Specific subreddit*
    - Post order*
    - Comment order*
    - Command keyword*
    - returns prompt_comment*


Search prompter - using prompt_comment* it prompts both Google and ChatGpt for an answer and returns said answer
    - Google API
    - Open AI API
    - prompt_comment*
    - GPT word limit
    - Google word limit
    - returns prompt_answer(google_result, gpt_result)*

Main - main bot loop
    - RedditCrawler -> SearchPrompter -> RedditCrawler ->... .
