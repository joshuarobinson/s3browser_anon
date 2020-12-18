## We specify the base image we need for our
## go application
FROM golang:1.15.2-alpine3.12 AS builder

RUN apk add build-base git musl-dev

## We create an /app directory within our image that will hold our application source files
RUN mkdir /app

RUN go get -u github.com/aws/aws-sdk-go/...

## We copy everything in the root directory into our /app directory
ADD . /app

## We specify that we now wish to execute  any further commands inside our /app directory
WORKDIR /app

## we run go build to compile the binary executable of our Go program
RUN go build -tags musl -o main .

## Our start command which kicks off our newly created binary executable
CMD ["/app/main"]
