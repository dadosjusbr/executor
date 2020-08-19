FROM golang:1.14.0-alpine

# Create output folder
RUN mkdir /output

# Move to working directory /app
WORKDIR /app

# Copy the code into the container
COPY . .

# Set necessary environmet variables needed
ENV GO111MODULE=on 

# Install package status
RUN apk update && apk add git && go get "github.com/dadosjusbr/executor/status"

# Build the application
RUN go build -o main

ENTRYPOINT ["./main"]
