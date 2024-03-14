package main

import (
	"github.com/cristalhq/jwt/v3"
)

type UserConfig struct {
	NamingScheme    string `json:"naming_scheme"`
	Embedded        bool   `json:"embedded"`
	UseJaas         bool   `json:"use_jaas"`
	ShowPrejoinPage bool   `json:"show_prejoin_page"`
}

type User struct {
	Avatar string `json:"avatar"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	ID     string `json:"id"`
}

type Context struct {
	User  User   `json:"user"`
	Group string `json:"group"`
}

type EnrichMeetingJwtRequest struct {
	Jwt string `json:"jwt"`
}

// Claims extends cristalhq/jwt standard claims to add jitsi-web-token specific fields
type Claims struct {
	jwt.StandardClaims
	Context Context `json:"context"`
	Room    string  `json:"room,omitempty"`
}

type JaaSUser struct {
	Avatar    string `json:"avatar"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	ID        string `json:"id,omitempty"`
	Moderator string `json:"moderator,omitempty"`
}

type JaaSFeatures struct {
	LiveStreaming string `json:"livestreaming"`
	Recording     string `json:"recording"`
	OutboundCall  string `json:"outbound-call"`
	Transcription string `json:"transcription"`
}

type JaaSContext struct {
	User     JaaSUser     `json:"user"`
	Features JaaSFeatures `json:"features"`
}

type JaaSClaims struct {
	Context  JaaSContext `json:"context"`
	Audience string      `json:"aud,omitempty"`
	Subject  string      `json:"sub,omitempty"`
	Issuer   string      `json:"iss,omitempty"`
	Room     string      `json:"room,omitempty"`
	Exp      int64       `json:"exp,omitempty"`
	Nbf      int64       `json:"nbf,omitempty"`
}

type JaaSSettings struct {
	Jwt  string `json:"jaasJwt"`
	Room string `json:"jaasRoom"`
}
