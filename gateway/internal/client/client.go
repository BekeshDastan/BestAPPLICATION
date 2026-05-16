package client

import (
	chatv1 "github.com/bekesh/social/gen/go/chat/v1"
	postv1 "github.com/bekesh/social/gen/go/post/v1"
	storyv1 "github.com/bekesh/social/gen/go/story/v1"
	userv1 "github.com/bekesh/social/gen/go/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Clients struct {
	User  userv1.UserServiceClient
	Post  postv1.PostServiceClient
	Chat  chatv1.ChatServiceClient
	Story storyv1.StoryServiceClient
}

func New(userAddr, postAddr, chatAddr, storyAddr string) (*Clients, func(), error) {
	userConn, err := grpc.NewClient("passthrough:///"+userAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	postConn, err := grpc.NewClient("passthrough:///"+postAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		userConn.Close()
		return nil, nil, err
	}
	chatConn, err := grpc.NewClient("passthrough:///"+chatAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		userConn.Close()
		postConn.Close()
		return nil, nil, err
	}
	storyConn, err := grpc.NewClient("passthrough:///"+storyAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		userConn.Close()
		postConn.Close()
		chatConn.Close()
		return nil, nil, err
	}

	cleanup := func() {
		userConn.Close()
		postConn.Close()
		chatConn.Close()
		storyConn.Close()
	}

	return &Clients{
		User:  userv1.NewUserServiceClient(userConn),
		Post:  postv1.NewPostServiceClient(postConn),
		Chat:  chatv1.NewChatServiceClient(chatConn),
		Story: storyv1.NewStoryServiceClient(storyConn),
	}, cleanup, nil
}
