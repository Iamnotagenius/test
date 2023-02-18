// Package server contains gRPC server implementation
package server

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/Iamnotagenius/test/db/service"
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
)

// DatabaseTestServer is gRPC server implementation of database service
type DatabaseTestServer struct {
	db *pg.DB
	service.UnimplementedDatabaseTestServer
}

// ErrUserNotFound is returned by GetUserByID when user with given ID does not exist
var ErrUserNotFound = errors.New("User not found")

func initDb(connOpts *pg.Options) (*pg.DB, error) {
	db := pg.Connect(connOpts)
	models := []interface{}{
		(*service.User)(nil),
	}

	for _, model := range models {
		err := db.Model(model).CreateTable(&orm.CreateTableOptions{
			Temp:        false,
			IfNotExists: true,
		})
		if err != nil {
			return nil, err
		}
	}
	return db, nil
}

// NewDatabaseServer creates new server instance
func NewDatabaseServer(connOpts *pg.Options) *DatabaseTestServer {
	db, err := initDb(connOpts)
	if err != nil {
		panic(err)
	}
	return &DatabaseTestServer{db: db}
}

// AddOrUpdateUser adds user to database if user's ID didn't exist, updates fields otherwise
func (s *DatabaseTestServer) AddOrUpdateUser(ctx context.Context, user *service.User) (*service.UpdateResponse, error) {
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

// GetUserByID retrieves user from database with given ID
func (s *DatabaseTestServer) GetUserByID(ctx context.Context, req *service.UserByIDRequest) (*service.User, error) {
	user := &service.User{Id: req.GetId()}
	err := s.db.ModelContext(ctx, user).Select()
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

// SearchUsersByName searches users in database by part of a name
func (s *DatabaseTestServer) SearchUsersByName(req *service.SearchByNameRequest, stream service.DatabaseTest_SearchUsersByNameServer) error {
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
