build:
	GOOS=linux go build -o ./bin/handler main.go

deploy: build
	serverless deploy

clean:
	if [ -d ./.serverless ] ; then rm -r ./.serverless ; fi
	if [ -d ./bin ] ; then rm -r ./bin ; fi

remove: clean
	serverless remove