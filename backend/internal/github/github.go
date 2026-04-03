package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/lucciano/prds/pkg/config"
)

const (
	authURL     = "https://github.com/login/oauth/authorize"
	tokenURL    = "https://github.com/login/oauth/access_token"
	apiBase     = "https://api.github.com"
)

type Service struct {
	cfg *config.Config
}

func NewService(cfg *config.Config) *Service {
	return &Service{cfg: cfg}
}

// AuthURL returns the GitHub OAuth2 authorization URL.
func (s *Service) AuthURL() string {
	params := url.Values{
		"client_id":    {s.cfg.GitHubClientID},
		"redirect_uri": {s.cfg.GitHubCallbackURL},
		"scope":        {"read:user,repo"},
	}
	return authURL + "?" + params.Encode()
}

// ExchangeCode exchanges an OAuth code for an access token.
func (s *Service) ExchangeCode(ctx context.Context, code string) (string, error) {
	data := url.Values{
		"client_id":     {s.cfg.GitHubClientID},
		"client_secret": {s.cfg.GitHubClientSecret},
		"code":          {code},
	}
	req, _ := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Error != "" {
		return "", fmt.Errorf("github oauth: %s", result.Error)
	}
	return result.AccessToken, nil
}

// GetUser fetches the authenticated GitHub user.
func (s *Service) GetUser(ctx context.Context, token string) (*GitHubUser, error) {
	u := &GitHubUser{}
	_, err := s.get(ctx, token, "/user", u)
	return u, err
}

// GetPullRequests fetches merged PRs for the authenticated user.
func (s *Service) GetPullRequests(ctx context.Context, token string) ([]PullRequest, error) {
	// Search API: merged PRs by user
	query := "is:pr+is:merged+author:@me"
	endpoint := fmt.Sprintf("/search/issues?q=%s&per_page=50", url.QueryEscape(query))

	var result struct {
		Items []PullRequest `json:"items"`
	}
	if _, err := s.get(ctx, token, endpoint, &result); err != nil {
		return nil, err
	}
	return result.Items, nil
}

// GetContributionData fetches event-based contribution data.
func (s *Service) GetContributionData(ctx context.Context, token string) (map[string]int, error) {
	var events []struct {
		Type      string `json:"type"`
		CreatedAt string `json:"created_at"`
	}
	if _, err := s.get(ctx, token, "/users/@me/events?per_page=100", &events); err != nil {
		return nil, err
	}
	counts := make(map[string]int)
	for _, e := range events {
		if len(e.CreatedAt) >= 10 {
			day := e.CreatedAt[:10]
			counts[day]++
		}
	}
	return counts, nil
}

func (s *Service) get(ctx context.Context, token, endpoint string, out interface{}) (interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", apiBase+endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("github API error: %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return nil, err
	}
	return out, nil
}

// ─── Types ────────────────────────────────────────────────────────────────────

type GitHubUser struct {
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	PublicRepos int  `json:"public_repos"`
	Followers int   `json:"followers"`
}

type PullRequest struct {
	Title      string `json:"title"`
	Body       string `json:"body"`
	HTMLURL    string `json:"html_url"`
	State      string `json:"state"`
	RepositoryURL string `json:"repository_url"`
	CreatedAt  string `json:"created_at"`
	MergedAt   string `json:"closed_at"`
}
