package tollbit

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	secretKey      string
	organizationId string
	userAgent      string

	httpClient *http.Client
}

type Options struct {
	HttpClient *http.Client
}

func NewClient(apiToken string, organizationId string, userAgent string, options ...func(*Options)) (*Client, error) {
	opts := Options{
		HttpClient: http.DefaultClient,
	}

	for _, fn := range options {
		fn(&opts)
	}

	c := &Client{
		secretKey:      apiToken,
		organizationId: organizationId,
		userAgent:      userAgent,

		httpClient: opts.HttpClient,
	}

	return c, nil
}

// LicenseType these are "enums" for the types of licenses
type LicenseType string

const (
	OnDemandLicense LicenseType = "ON_DEMAND_LICENSE"
)

type tokenStruct struct {
	OrgCuid        string      `json:"orgCuid"`
	Key            string      `json:"key"`
	Url            string      `json:"url"`
	UserAgent      string      `json:"userAgent"`
	MaxPriceMicros int64       `json:"maxPriceMicros"`
	Currency       string      `json:"currency"`
	LicenseType    LicenseType `json:"licenseType"`
}

type TokenParams struct {
	Url            string
	MaxPriceMicros int64
	Currency       string // only supports USD for now
	LicenseType    LicenseType
}

type ContentResponse struct {
	Content  Content      `json:"content"`
	Metadata string       `json:"metadata"`
	Rate     RateResponse `json:"rate"`
}

type Content struct {
	Header string `json:"header"`
	Main   string `json:"main"`
	Footer string `json:"footer"`
}

type RateResponse struct {
	PriceMicros int64  `json:"priceMicros"`
	Currency    string `json:"currency"`
	LicenseType string `json:"licenseType"`
	LicensePath string `json:"licensePath"`
	Error       string `json:"error"`
}

func (c *Client) GenerateToken(params TokenParams) (string, error) {
	token := tokenStruct{
		OrgCuid:        c.organizationId,
		Key:            c.secretKey,
		Url:            params.Url,
		UserAgent:      c.userAgent,
		MaxPriceMicros: params.MaxPriceMicros,
		Currency:       params.Currency,
		LicenseType:    params.LicenseType,
	}
	jsonToken, err := json.Marshal(token)
	if err != nil {
		return "", err
	}
	encryptedToken, err := Encrypt(jsonToken, c.secretKey)
	if err != nil {
		return "", err
	}
	return encryptedToken, nil
}

func (c *Client) GetContentWithToken(ctx context.Context, token string) (ContentResponse, error) {
	decryptedToken, err := Decrypt(token, c.secretKey)
	var t tokenStruct
	err = json.Unmarshal(decryptedToken, &t)
	if err != nil {
		return ContentResponse{}, err
	}

	// first remove the https:// part
	tollbitUrl := strings.TrimPrefix(strings.TrimPrefix(t.Url, "https://"), "http://")
	// then remove any potential www. part
	tollbitUrl = strings.TrimPrefix(tollbitUrl, "www.")
	// then construct the actual url
	tollbitUrl = "https://api.tollbit.com/dev/v1/content/" + tollbitUrl
	req, err := http.NewRequestWithContext(ctx, "GET", tollbitUrl, nil)
	if err != nil {
		return ContentResponse{}, err
	}
	req.Header.Add("TollbitOrgCuid", t.OrgCuid)
	req.Header.Add("User-Agent", "Mozilla/5.0 (compatible; "+c.userAgent+"; +https://tollbit.com/bot)")
	req.Header.Add("TollbitToken", token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ContentResponse{}, err
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	var contentResponse []ContentResponse
	err = json.Unmarshal(bodyBytes, &contentResponse)
	if err != nil {
		return ContentResponse{}, err
	}
	if len(contentResponse) == 0 || contentResponse[0].Content.Main == "" {
		return ContentResponse{}, errors.New("could not get content")
	}
	return contentResponse[0], nil
}

func (c *Client) GetContent(ctx context.Context, params TokenParams) (ContentResponse, error) {
	token, err := c.GenerateToken(params)
	if err != nil {
		return ContentResponse{}, err
	}
	return c.GetContentWithToken(ctx, token)
}

func (c *Client) GetRate(ctx context.Context, targetUrl string) (RateResponse, error) {
	// first remove the https:// part
	tollbitUrl := strings.TrimPrefix(strings.TrimPrefix(targetUrl, "https://"), "http://")
	// then remove any potential www. part
	tollbitUrl = strings.TrimPrefix(tollbitUrl, "www.")
	// then construct the actual url
	tollbitUrl = "https://api.tollbit.com/dev/v1/rate/" + tollbitUrl
	req, err := http.NewRequestWithContext(ctx, "GET", tollbitUrl, nil)
	if err != nil {
		return RateResponse{}, err
	}
	req.Header.Add("User-Agent", "Mozilla/5.0 (compatible; "+c.userAgent+"; +https://tollbit.com/bot)")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return RateResponse{}, err
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	var rateResponse []RateResponse
	err = json.Unmarshal(bodyBytes, &rateResponse)
	if err != nil {
		return RateResponse{}, err
	}
	if len(rateResponse) == 0 {
		return RateResponse{}, errors.New("could not get rate")
	}
	return rateResponse[0], nil
}
