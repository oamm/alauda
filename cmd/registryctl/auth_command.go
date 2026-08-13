package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

func newAuthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication users and API tokens",
	}
	cmd.AddCommand(newAuthLoginCommand())
	cmd.AddCommand(newAuthMeCommand())
	cmd.AddCommand(newAuthUserCreateCommand())
	cmd.AddCommand(newAuthUserListCommand())
	cmd.AddCommand(newAuthTokenCreateCommand())
	cmd.AddCommand(newAuthTokenListCommand())
	cmd.AddCommand(newAuthTokenRevokeCommand())
	return cmd
}

func newAuthLoginCommand() *cobra.Command {
	var username, password string
	c := &cobra.Command{
		Use:   "login",
		Short: "Create a bearer session token",
		RunE: func(cmd *cobra.Command, args []string) error {
			if username == "" || password == "" {
				return fmt.Errorf("--username and --password are required")
			}
			payload, _ := json.Marshal(map[string]string{"username": username, "password": password})
			var response map[string]any
			if err := doJSON(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(payload), &response); err != nil {
				return err
			}
			printAny(response)
			return nil
		},
	}
	c.Flags().StringVar(&username, "username", "", "Username")
	c.Flags().StringVar(&password, "password", "", "Password")
	return c
}

func newAuthMeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Show the authenticated user",
		RunE: func(cmd *cobra.Command, args []string) error {
			var response map[string]any
			if err := doJSON(http.MethodGet, "/api/v1/auth/me", nil, &response); err != nil {
				return err
			}
			printAny(response)
			return nil
		},
	}
}

func newAuthUserCreateCommand() *cobra.Command {
	var username, email, displayName, password, role string
	c := &cobra.Command{
		Use:   "user-create",
		Short: "Create a local user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if username == "" || email == "" || displayName == "" || password == "" || role == "" {
				return fmt.Errorf("--username, --email, --display-name, --password, and --role are required")
			}
			payload, _ := json.Marshal(map[string]string{
				"username":    username,
				"email":       email,
				"displayName": displayName,
				"password":    password,
				"role":        role,
			})
			var response map[string]any
			if err := doJSON(http.MethodPost, "/api/v1/auth/users", bytes.NewReader(payload), &response); err != nil {
				return err
			}
			printAny(response)
			return nil
		},
	}
	c.Flags().StringVar(&username, "username", "", "Username")
	c.Flags().StringVar(&email, "email", "", "Email")
	c.Flags().StringVar(&displayName, "display-name", "", "Display name")
	c.Flags().StringVar(&password, "password", "", "Password")
	c.Flags().StringVar(&role, "role", "Viewer", "Role: Administrator|Operator|Viewer|Automation")
	return c
}

func newAuthUserListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "user-list",
		Short: "List local users",
		RunE: func(cmd *cobra.Command, args []string) error {
			var response map[string]any
			if err := doJSON(http.MethodGet, "/api/v1/auth/users", nil, &response); err != nil {
				return err
			}
			printAny(response)
			return nil
		},
	}
}

func newAuthTokenCreateCommand() *cobra.Command {
	var userID, name, expiresIn string
	var scopes []string
	c := &cobra.Command{
		Use:   "token-create",
		Short: "Create an API token",
		RunE: func(cmd *cobra.Command, args []string) error {
			if userID == "" || name == "" {
				return fmt.Errorf("--user-id and --name are required")
			}
			payloadMap := map[string]any{"userId": userID, "name": name, "scopes": scopes}
			if expiresIn != "" {
				duration, err := time.ParseDuration(expiresIn)
				if err != nil {
					return err
				}
				expiresAt := time.Now().UTC().Add(duration)
				payloadMap["expiresAt"] = expiresAt
			}
			payload, _ := json.Marshal(payloadMap)
			var response map[string]any
			if err := doJSON(http.MethodPost, "/api/v1/auth/tokens", bytes.NewReader(payload), &response); err != nil {
				return err
			}
			printAny(response)
			return nil
		},
	}
	c.Flags().StringVar(&userID, "user-id", "", "User ID")
	c.Flags().StringVar(&name, "name", "", "Token name")
	c.Flags().StringSliceVar(&scopes, "scope", []string{"read"}, "Token scopes: read,write,admin")
	c.Flags().StringVar(&expiresIn, "expires-in", "", "Expiration duration, for example 24h or 720h")
	return c
}

func newAuthTokenListCommand() *cobra.Command {
	var userID string
	c := &cobra.Command{
		Use:   "token-list",
		Short: "List API tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "/api/v1/auth/tokens"
			if userID != "" {
				path += "?userId=" + userID
			}
			var response map[string]any
			if err := doJSON(http.MethodGet, path, nil, &response); err != nil {
				return err
			}
			printAny(response)
			return nil
		},
	}
	c.Flags().StringVar(&userID, "user-id", "", "User ID")
	return c
}

func newAuthTokenRevokeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "token-revoke <token-id>",
		Short: "Revoke an API token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := doJSON(http.MethodDelete, "/api/v1/auth/tokens/"+args[0], nil, nil); err != nil {
				return err
			}
			fmt.Println("revoked")
			return nil
		},
	}
}
