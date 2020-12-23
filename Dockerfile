FROM golang:1.15.6-alpine3.12 AS builder

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

FROM golang:1.15.6-alpine3.12
COPY --from=builder /app/favicon.ico /app/favicon.ico
COPY --from=builder /app/main /app/main

## Our start command which kicks off our newly created binary executable
CMD ["/app/main"]
