// REST API for admins
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/Iamnotagenius/test/db/service"
	"github.com/coreos/go-oidc"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type handler struct {
	service.DatabaseTestClient
	authChan chan int64
	provider *oidc.Provider
	sessions map[string]int64
	state    string
}

var (
	grpcDbServiceAddress = flag.String("db-service-addr", "localhost:50051", "Address of grpc DB service")

	oauth2Config = oauth2.Config{
		ClientID:     os.Getenv("ITMOID_CLIENT_ID"),
		ClientSecret: os.Getenv("ITMOID_CLIENT_SECRET"),
		Scopes:       []string{oidc.ScopeOpenID},
		RedirectURL:  "http://localhost:8080/",
	}
)

func (handler *handler) authenticate(ctx *gin.Context) {
	code, ok := ctx.GetQuery("code")
	if !ok {
		return
	}

	token, err := oauth2Config.Exchange(ctx, code)
	if err != nil {
		log.Printf("Exchange failed: %v", err)
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		log.Panicln("Missing token")
		return
	}
	state, _ := ctx.GetQuery("state")
	if handler.state != state {
		log.Println("States did not match. CSRF attack?")
		return
	}

	idToken, err := handler.provider.Verifier(
		&oidc.Config{ClientID: os.Getenv("ITMOID_CLIENT_ID")}).Verify(ctx, rawIDToken)
	if err != nil {
		log.Printf("Token parse failed: %v", err)
		return
	}

	var claims struct {
		Sub string `json:"sub"`
		Isu int64  `json:"isu"`
	}
	if err := idToken.Claims(&claims); err != nil {
		log.Printf("Token unmarshal failed: %v", err)
		return
	}

	handler.sessions[ctx.RemoteIP()] = claims.Isu
	log.Printf("Added session for IP %v with ISU %v", ctx.RemoteIP(), claims.Isu)
	ctx.IndentedJSON(http.StatusOK, gin.H{"status": fmt.Sprintf("Successfully authenticated: %v", claims.Isu)})
}

func generateState() string {
	buffer := make([]byte, 25)
	if _, err := io.ReadFull(rand.Reader, buffer); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(buffer)
}

func (handler *handler) authorize(ctx *gin.Context) {
	isu, ok := handler.sessions[ctx.RemoteIP()]
	if !ok {
		log.Println("Redirect")
		ctx.Redirect(http.StatusFound, oauth2Config.AuthCodeURL(handler.state))
		return
	}

	user, err := handler.GetUserByID(ctx, &service.UserByIDRequest{Id: isu})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			respondWithError(ctx, http.StatusForbidden, "User not found in database")
			return
		}
	}

	switch user.GetRole() {
	case service.Role_ROLE_UNSPECIFIED:
	case service.Role_ROLE_USER:
		respondWithError(ctx, http.StatusForbidden, "User does not have permssion to use this API")
		return
	}

	ctx.Set("user_id", isu)

	switch ctx.Request.Method {
	case "GET":
		ctx.Next()
		return
	case "POST":
		if user.GetRole() == service.Role_ROLE_READ_ONLY_ADMIN {
			respondWithError(ctx, http.StatusForbidden, "Read-only admins cannot do POST requests")
			return
		}
		ctx.Next()
		return
	}

	respondWithError(ctx, http.StatusForbidden, "Unsupported method: %v", ctx.Request.Method)
}

func (handler *handler) getUsers(ctx *gin.Context) {
	q, _ := ctx.GetQuery("query")
	stream, err := handler.SearchUsersByName(ctx, &service.SearchByNameRequest{Query: q})
	if err != nil {
		respondWithError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	users := make([]*service.User, 0)
	for {
		user, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			respondWithError(ctx, http.StatusInternalServerError, err.Error())
			return
		}
		users = append(users, user)
	}

	ctx.Header("Content-Type", "application/json")
	ctx.IndentedJSON(http.StatusOK, &users)
}

func (handler *handler) getUser(ctx *gin.Context) {
	user, err := handler.getUserFromParam(ctx)
	if err != nil {
		return
	}
	ctx.Header("Content-Type", "application/json")
	ctx.IndentedJSON(http.StatusOK, user)
}

func (handler *handler) changeUserFields(ctx *gin.Context) {
	user, err := handler.getUserFromParam(ctx)
	if err != nil {
		return
	}
	oldRole := user.GetRole()
	err = ctx.BindJSON(&user)
	if err != nil {
		respondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if oldRole != user.GetRole() {
		userID, _ := ctx.Get("user_id")
		id, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)
		if id == userID.(int64) {
			respondWithError(ctx, http.StatusBadRequest, "Admins cannot change their role")
			return
		}
	}

	handler.AddOrUpdateUser(ctx, user)
}

func (handler *handler) getUserFromParam(ctx *gin.Context) (*service.User, error) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		respondWithError(ctx, http.StatusBadRequest, "Id has wrong format")
		return nil, err
	}

	user, err := handler.GetUserByID(ctx, &service.UserByIDRequest{Id: id})
	if status.Code(err) == codes.NotFound {
		respondWithError(ctx, http.StatusNotFound, "User with id %v not found", id)
		return nil, err
	}
	if err != nil {
		respondWithError(ctx, http.StatusInternalServerError, err.Error())
		return nil, err
	}

	return user, nil
}

func respondWithError(c *gin.Context, code int, format string, args ...interface{}) {
	c.AbortWithStatusJSON(code, gin.H{"error": fmt.Sprintf(format, args...)})
}

func main() {
	router := gin.Default()
	grpcConn, err := grpc.Dial(*grpcDbServiceAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Panicf("Failed to establish connection with database service: %v", err)
	}
	defer grpcConn.Close()
	provider, err := oidc.NewProvider(context.Background(), "https://id.itmo.ru/auth/realms/itmo")
	if err != nil {
		log.Panicf("Invalid provider: %v", err)
	}
	oauth2Config.Endpoint = provider.Endpoint()
	handler := handler{
		DatabaseTestClient: nil,
		authChan:           make(chan int64),
		provider:           provider,
		sessions:           map[string]int64{},
		state:              generateState(),
	}

	router.GET("/", handler.authenticate)
	router.Use(handler.authorize)
	router.GET("/users", handler.getUsers)
	router.GET("/users/:id", handler.getUser)
	router.POST("/users/:id", handler.changeUserFields)
	router.Run(":8080")
}
