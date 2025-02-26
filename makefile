
# run all unit tests 
test:
	go test ./...

# run all tests that require spinning up minio or testing against real endpoints
# these are expected to be slower
teste2e:
	go test -tags 'e2e' ./... 

# check for unused code in the repo
deadcode:
	deadcode -tags "e2e" -test ./... 