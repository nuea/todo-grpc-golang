# todo-grpc-golang

## Getting Started
Getting up and running is as easy as 1, 2, 3.
1. Make sure you have [GO](https://go.dev/dl/) and [Protocol Buffers](https://grpc.io/docs/protoc-installation/) installed.<br>
    * Can check GO version: <br> 
    ``` 
        go version
    ```
    * Can check Protocol Buffers version: <br>
    ```
        protoc --version
    ```
2. Go plugins for the protocol compiler:
    * Install the protocol compiler plugins for Go using the following commands:
    ```
        go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
        go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
    ```
    * Update your PATH so that the protoc compiler can find the plugins:
    ```
        export PATH="$PATH:$(go env GOPATH)/bin"
    ```
3. Use __"Makefile"__ for generate gRPC and build:
    * Can use command for Display info related to the build:
    ```
        make about
    ```

