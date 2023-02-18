package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Iamnotagenius/test/db/service"
	"github.com/go-pg/pg/v10"
)

type databaseTestServer struct {
	db *pg.DB
	service.UnimplementedDatabaseTestServer
}

func NewDatabaseServer(connOpts *pg.Options) *databaseTestServer {
	db, err := InitDb(connOpts)
	if err != nil {
		panic(err)
	}
	return &databaseTestServer{db: db}
}

func (s *databaseTestServer) AddOrUpdateUser(ctx context.Context, user *service.User) (*service.UpdateResponse, error) {
	query := s.db.ModelContext(ctx, user).WherePK()
	if exists, _ := query.Exists(); !exists {
		_, err := s.db.ModelContext(ctx, user).Insert()
		if err != nil {
			return nil, err
		}
		return &service.UpdateResponse{}, nil
	}

	query.Update()
	return &service.UpdateResponse{}, nil
}

func (s *databaseTestServer) GetUserById(ctx context.Context, req *service.UserByIdRequest) (*service.User, error) {
	user := &service.User{Id: req.GetId()}
	err := s.db.ModelContext(ctx, user).Select()
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *databaseTestServer) SearchUsersByName(req *service.SearchByNameRequest, stream service.DatabaseTest_SearchUsersByNameServer) error {
	var users []*service.User
	s.db.Model(&users).Where(fmt.Sprintf("name LIKE '%%%v%%'", req.Query)).Select()
	for _, user := range users {
		err := stream.Send(user)
		if err != nil {
			log.Printf("Error while executing query: %v", err)
			return err
		}
	}
	return nil
}
