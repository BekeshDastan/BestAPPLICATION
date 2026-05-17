package client

import (
	chatv1  "github.com/bekesh/social/gen/go/chat/v1"
	notifv1 "github.com/bekesh/social/gen/go/notification/v1"
	postv1  "github.com/bekesh/social/gen/go/post/v1"
	storyv1 "github.com/bekesh/social/gen/go/story/v1"
	userv1  "github.com/bekesh/social/gen/go/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Clients struct {
	User         userv1.UserServiceClient
	Post         postv1.PostServiceClient
	Chat         chatv1.ChatServiceClient
	Story        storyv1.StoryServiceClient
	Notification notifv1.NotificationServiceClient
}

func New(userAddr, postAddr, chatAddr, storyAddr, notifAddr string) (*Clients, func(), error) {
	dial := func(addr string) (*grpc.ClientConn, error) {
		return grpc.NewClient("passthrough:///"+addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	userConn, err := dial(userAddr)
	if err != nil {
		return nil, nil, err
	}
	postConn, err := dial(postAddr)
	if err != nil {
		userConn.Close()
		return nil, nil, err
	}
	chatConn, err := dial(chatAddr)
	if err != nil {
		userConn.Close(); postConn.Close()
		return nil, nil, err
	}
	storyConn, err := dial(storyAddr)
	if err != nil {
		userConn.Close(); postConn.Close(); chatConn.Close()
		return nil, nil, err
	}
	notifConn, err := dial(notifAddr)
	if err != nil {
		userConn.Close(); postConn.Close(); chatConn.Close(); storyConn.Close()
		return nil, nil, err
	}

	cleanup := func() {
		userConn.Close(); postConn.Close(); chatConn.Close()
		storyConn.Close(); notifConn.Close()
	}

	return &Clients{
		User:         userv1.NewUserServiceClient(userConn),
		Post:         postv1.NewPostServiceClient(postConn),
		Chat:         chatv1.NewChatServiceClient(chatConn),
		Story:        storyv1.NewStoryServiceClient(storyConn),
		Notification: notifv1.NewNotificationServiceClient(notifConn),
	}, cleanup, nil
}
