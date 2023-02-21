package main

import (
	"context"
	"fmt"
	"log"
	"net"

	pb "github.com/nuea/todo-grpc-golang/todo/proto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	pb.TodoServiceServer
}

type Todo struct {
	Id          primitive.ObjectID `bson:"_id,omitempty"`
	Title       string             `bson:"title"`
	Description string             `bson:"description"`
	Status      bool               `bson:"status"`
}

func documentToTodo(data *Todo) *pb.Todo {
	return &pb.Todo{
		Id:          data.Id.Hex(),
		Title:       data.Title,
		Description: data.Description,
		Status:      data.Status,
	}
}

func (s *Server) CreateTodo(ctx context.Context, req *pb.Todo) (*pb.TodoResponse, error) {
	fmt.Printf("Create todo invoked with: %v\n", req)
	todo := Todo{
		Title:       req.GetTitle(),
		Description: req.GetDescription(),
		Status:      req.GetStatus(),
	}

	res, err := collection.InsertOne(ctx, todo)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal Error: %v\n", err),
		)
	}

	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Cannot convert to ObjectId"),
		)
	}

	return &pb.TodoResponse{
		Todo: &pb.Todo{
			Id:          oid.Hex(),
			Title:       todo.Title,
			Description: todo.Description,
			Status:      todo.Status,
		},
	}, nil
}

func (s *Server) ReadTodo(ctx context.Context, req *pb.TodoId) (*pb.TodoResponse, error) {
	fmt.Printf("Read todo invoked with: %v\n", req)
	todoId := req.GetId()

	oid, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("Cannot parse Id: %v", req.Id),
		)
	}
	data := &Todo{}
	filter := bson.M{"_id": oid}
	res := collection.FindOne(ctx, filter)

	if err := res.Decode(data); err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Cannot find todo with specified id: %v", todoId),
		)
	}

	return &pb.TodoResponse{
		Todo: documentToTodo(data),
	}, nil
}

func (s *Server) UpdateTodo(ctx context.Context, req *pb.Todo) (*pb.TodoResponse, error) {
	fmt.Printf("Update todo was invoked with %v\n", req)

	oid, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("Cannot parse Id: %v", req.Id),
		)
	}
	data := &Todo{}
	filter := bson.M{"_id": oid}
	res := collection.FindOne(ctx, filter)

	if err := res.Decode(data); err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Cannot find todo with specified id: %v", req.Id),
		)
	}
	data.Title = req.GetTitle()
	data.Description = req.GetDescription()
	data.Status = req.GetStatus()

	resUpdate, updateErr := collection.ReplaceOne(ctx, filter, data)
	if updateErr != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Cannot update data: %v", updateErr),
		)
	}
	if resUpdate.MatchedCount == 0 {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Cannot find todo with Id: %v", req.Id),
		)
	}

	return &pb.TodoResponse{
		Todo: documentToTodo(data),
	}, nil
}

func (s *Server) DeleteTodo(ctx context.Context, req *pb.TodoId) (*emptypb.Empty, error) {
	fmt.Printf("Delete todo was invoked with %v\n", req)

	oid, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("Cannot parse Id: %v", req.Id),
		)
	}

	filter := bson.M{"_id": oid}
	res, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Cannot delete todo: %v", err),
		)
	}

	if res.DeletedCount == 0 {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Cannot find todo with Id: %v", req.Id),
		)
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) ListTodos(req *emptypb.Empty, stream pb.TodoService_ListTodosServer) error {
	fmt.Println("List todo was invoked")

	cur, err := collection.Find(context.Background(), primitive.D{{}})
	if err != nil {
		return status.Errorf(
			codes.Internal,
			fmt.Sprintf("Unknown internal error: %v", err),
		)
	}
	defer cur.Close(context.Background())
	for cur.Next(context.Background()) {
		data := &Todo{}
		err := cur.Decode(data)
		if err != nil {
			return status.Errorf(
				codes.Internal,
				fmt.Sprintf("Error while decoding data : %v", err),
			)

		}
		stream.Send(&pb.TodoResponse{Todo: documentToTodo(data)})
	}
	if err := cur.Err(); err != nil {
		return status.Errorf(
			codes.Internal,
			fmt.Sprintf("Unknown internal error: %v", err),
		)
	}

	return nil
}

var collection *mongo.Collection

const addr string = "localhost:50051"
const mongoURL = "mongodb://localhost:27017"

func main() {
	fmt.Println("Connect to MongoDB")
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURL))
	if err != nil {
		log.Fatal(err)
	}
	err = client.Connect(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("TODO List Service Started")
	collection = client.Database("mydb").Collection("todolist")

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	fmt.Printf("Listening on %s\n", addr)

	var opts []grpc.ServerOption
	s := grpc.NewServer(opts...)
	pb.RegisterTodoServiceServer(s, &Server{})
	// Register reflection service on gRPC server.
	reflection.Register(s)

	if err = s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v\n", err)
	}
}
