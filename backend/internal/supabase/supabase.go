package supabase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"rpms-backend/internal/config"
)

type Client struct {
	config *config.Config
}

type SupabaseError struct {
	StatusCode int
	Message    string
}

func (e *SupabaseError) Error() string {
	return fmt.Sprintf("supabase error (status %d): %s", e.StatusCode, e.Message)
}

func NewClient(cfg *config.Config) *Client {
	return &Client{config: cfg}
}

type SignUpRequest struct {
	Email    string                 `json:"email"`
	Password string                 `json:"password"`
	Data     map[string]interface{} `json:"data"`
}

type User struct {
	ID           string                 `json:"id"`
	Email        string                 `json:"email"`
	UserMetadata map[string]interface{} `json:"user_metadata"`
}

type SignUpResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func (s *Client) SignUp(email, password string, data map[string]interface{}) (*SignUpResponse, error) {
	url := fmt.Sprintf("%s/auth/v1/signup", s.config.Supabase.URL)
	reqBody, _ := json.Marshal(SignUpRequest{
		Email:    email,
		Password: password,
		Data:     data,
	})

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("apikey", s.config.Supabase.AnonKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("supabase signup failed: %v", errResp)
	}

	var result SignUpResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

type VerifyRequest struct {
	Type  string `json:"type"`
	Email string `json:"email"`
	Token string `json:"token"`
}

type VerifyResponse struct {
	User User `json:"user"`
}

func (s *Client) Verify(email, token string) (*User, error) {
	url := fmt.Sprintf("%s/auth/v1/verify", s.config.Supabase.URL)
	reqBody, _ := json.Marshal(VerifyRequest{
		Type:  "signup",
		Email: email,
		Token: token,
	})

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("apikey", s.config.Supabase.AnonKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("supabase verify failed: %v", errResp)
	}

	var result VerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.User, nil
}

type ResendRequest struct {
	Type  string `json:"type"`
	Email string `json:"email"`
}

func (s *Client) Resend(email string) error {
	url := fmt.Sprintf("%s/auth/v1/resend", s.config.Supabase.URL)
	reqBody, _ := json.Marshal(ResendRequest{
		Type:  "signup",
		Email: email,
	})

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}

	req.Header.Set("apikey", s.config.Supabase.AnonKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)

		msg := "unknown error"
		if m, ok := errResp["msg"].(string); ok {
			msg = m
		} else if m, ok := errResp["error_description"].(string); ok {
			msg = m
		}

		return &SupabaseError{
			StatusCode: resp.StatusCode,
			Message:    msg,
		}
	}

	return nil
}

type SignInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SignInResponse struct {
	User         User   `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (s *Client) SignIn(email, password string) (*SignInResponse, error) {
	url := fmt.Sprintf("%s/auth/v1/token?grant_type=password", s.config.Supabase.URL)
	reqBody, _ := json.Marshal(SignInRequest{
		Email:    email,
		Password: password,
	})

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("apikey", s.config.Supabase.AnonKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("supabase signin failed: %v", errResp)
	}

	var result SignInResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

type AdminCreateUserRequest struct {
	Email        string                 `json:"email"`
	Password     string                 `json:"password"`
	EmailConfirm bool                   `json:"email_confirm"`
	UserMetadata map[string]interface{} `json:"user_metadata"`
	AppMetadata  map[string]interface{} `json:"app_metadata"`
}

func (s *Client) AdminCreateUser(email, password string, userMetadata map[string]interface{}) (*User, error) {
	url := fmt.Sprintf("%s/auth/v1/admin/users", s.config.Supabase.URL)
	reqBody, _ := json.Marshal(AdminCreateUserRequest{
		Email:        email,
		Password:     password,
		EmailConfirm: true,
		UserMetadata: userMetadata,
	})

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	// Use Service Role Key for Admin operations
	req.Header.Set("apikey", s.config.Supabase.ServiceRoleKey)
	req.Header.Set("Authorization", "Bearer "+s.config.Supabase.ServiceRoleKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("supabase admin create user failed (status %d): %v", resp.StatusCode, errResp)
	}

	var result User
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
