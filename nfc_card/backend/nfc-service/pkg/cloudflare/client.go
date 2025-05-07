package cloudflare

// Client Cloudflare客户端
type Client struct {
	APIKey string
	Email  string
	ZoneID string
}

// NewClient 创建新的Cloudflare客户端
func NewClient(apiKey, email, zoneID string) *Client {
	return &Client{
		APIKey: apiKey,
		Email:  email,
		ZoneID: zoneID,
	}
}
