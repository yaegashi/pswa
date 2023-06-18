package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

const (
	GraphMemberGroupsRequestURL  = "https://graph.microsoft.com/v1.0/me/getMemberObjects"
	GraphMemberGroupsRequestBody = `{"securityEnabledOnly":true}`
)

type GraphMemberGroupsResponse struct {
	Value []string `json:"value"`
}

func GraphMemberGroupsRequest(ctx context.Context, oauth2Token *oauth2.Token) ([]string, error) {
	reqBody := bytes.NewBufferString(GraphMemberGroupsRequestBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, GraphMemberGroupsRequestURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("GraphMemberGroupsRequest failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+oauth2Token.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GraphMemberGroupsRequest failed: %w", err)
	}
	defer res.Body.Close()
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("GraphMemberGroupsRequest failed: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GraphMemberGroupsRequest failed: %s: %s", res.Status, string(resBody))
	}
	var resGraph GraphMemberGroupsResponse
	err = json.Unmarshal(resBody, &resGraph)
	if err != nil {
		return nil, fmt.Errorf("GraphMemberGroupsRequest failed: %w: %s", err, string(resBody))
	}
	return resGraph.Value, nil
}
