package taiga

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

func (c *Client) ListMemberships(ctx context.Context, projectID int64) ([]Membership, error) {
	query := url.Values{"project": []string{fmt.Sprint(projectID)}, "page_size": []string{"1000"}}
	var memberships []Membership
	_, err := c.Get(ctx, "memberships", query, &memberships)
	return memberships, err
}

func (c *Client) CreateMembership(ctx context.Context, request CreateMembershipRequest) (Membership, error) {
	var membership Membership
	_, err := c.Post(ctx, "memberships", request, &membership)
	return membership, err
}

func (c *Client) UpdateMembership(ctx context.Context, id int64, request UpdateMembershipRequest) (Membership, error) {
	var membership Membership
	_, err := c.Patch(ctx, fmt.Sprintf("memberships/%d", id), request, &membership)
	return membership, err
}

func (c *Client) DeleteMembership(ctx context.Context, id int64) error {
	return c.Delete(ctx, fmt.Sprintf("memberships/%d", id))
}

func (c *Client) ListRoles(ctx context.Context, projectID int64) ([]Role, error) {
	query := url.Values{"project": []string{fmt.Sprint(projectID)}, "page_size": []string{"1000"}}
	var roles []Role
	_, err := c.Get(ctx, "roles", query, &roles)
	return roles, err
}

func (c *Client) CreateRole(ctx context.Context, request CreateRoleRequest) (Role, error) {
	var role Role
	_, err := c.Post(ctx, "roles", request, &role)
	return role, err
}

func (c *Client) UpdateRole(ctx context.Context, id int64, request UpdateRoleRequest) (Role, error) {
	var role Role
	_, err := c.Patch(ctx, fmt.Sprintf("roles/%d", id), request, &role)
	return role, err
}

func (c *Client) DeleteRole(ctx context.Context, id int64, moveTo *int64) error {
	query := url.Values{}
	if moveTo != nil {
		query.Set("moveTo", fmt.Sprint(*moveTo))
	}
	_, err := c.doJSON(ctx, http.MethodDelete, fmt.Sprintf("roles/%d", id), query, nil, nil)
	return err
}
