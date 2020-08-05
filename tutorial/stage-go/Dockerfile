FROM golang:1.14.0-alpine

# Create output folder
RUN mkdir /output

# Move to working directory /build
WORKDIR /app

# Copy the code into the container
COPY . .

# Build the application
RUN go build -o main

ENTRYPOINT ["./main"]
