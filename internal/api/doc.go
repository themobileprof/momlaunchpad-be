// Package api implements MomLaunchpad HTTP handlers (Gin).
//
// OpenAPI generation (future): install swag (`go install github.com/swaggo/swag/cmd/swag@latest`)
// and run from repo root: `swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal`.
// Handler annotations use swag comment tags; shared request/response schemas live in openapi_community.go.
//
// @title MomLaunchpad API
// @version 1.0
// @description REST API for the MomLaunchpad mobile app and admin tools.
// @host localhost:8080
// @BasePath /api
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT access token. Send as `Authorization: Bearer <token>`.
package api
