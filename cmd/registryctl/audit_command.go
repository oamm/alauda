package main

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

func newAuditCommand() *cobra.Command {
	var actor, action, resourceType, resourceID, environmentID string
	var pageSize int
	c := &cobra.Command{
		Use:   "audit-log",
		Short: "Query audit logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			values := url.Values{}
			if actor != "" {
				values.Set("actor", actor)
			}
			if action != "" {
				values.Set("action", action)
			}
			if resourceType != "" {
				values.Set("resourceType", resourceType)
			}
			if resourceID != "" {
				values.Set("resourceId", resourceID)
			}
			if environmentID != "" {
				values.Set("environmentId", environmentID)
			}
			if pageSize > 0 {
				values.Set("pageSize", strconv.Itoa(pageSize))
			}
			path := "/api/v1/audit-logs"
			if encoded := values.Encode(); encoded != "" {
				path += "?" + encoded
			}
			var response map[string]any
			if err := doJSON(http.MethodGet, path, nil, &response); err != nil {
				return err
			}
			printAny(response)
			return nil
		},
	}
	c.Flags().StringVar(&actor, "actor", "", "Filter by actor")
	c.Flags().StringVar(&action, "action", "", "Filter by action")
	c.Flags().StringVar(&resourceType, "resource-type", "", "Filter by resource type")
	c.Flags().StringVar(&resourceID, "resource-id", "", "Filter by resource ID")
	c.Flags().StringVar(&environmentID, "environment-id", "", "Filter by environment ID")
	c.Flags().IntVar(&pageSize, "page-size", 50, "Page size")
	return c
}
