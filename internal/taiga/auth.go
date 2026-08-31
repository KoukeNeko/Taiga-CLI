package taiga

import "context"

type LoginRequest struct {
	Type     string `json:"type"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (c *Client) Login(ctx context.Context, username, password string) (AuthResponse, error) {
	request := LoginRequest{Type: "normal", Username: username, Password: password}
	var response AuthResponse
	_, err := c.Post(ctx, "auth", request, &response)
	return response, err
}

func (c *Client) Me(ctx context.Context) (User, error) {
	var user User
	_, err := c.Get(ctx, "users/me", nil, &user)
	return user, err
}
