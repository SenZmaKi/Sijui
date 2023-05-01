module sijui

go 1.20

require crawler v0.0.0

require (
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/stretchr/testify v1.8.1 // indirect
	github.com/vartanbeno/go-reddit/v2 v2.0.1 // indirect
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/oauth2 v0.7.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)

replace crawler => ../crawler

replace searchAndPrompt => ../searchAndPrompt
