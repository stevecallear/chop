build:
	GOOS=linux go build -o handler main.go

pack: build
	if ! [ -d ./.package ]; then mkdir ./.package ; fi
	zip ./.package/handler.zip ./handler && rm ./handler

deploy: pack
	serverless deploy

clean:
	if [ -d ./.serverless ] ; then rm -r ./.serverless ; fi
	if [ -d ./.package ] ; then rm -r ./.package ; fi
	if [ -f ./handler ] ; then rm ./handler ; fi

remove: clean
	serverless remove