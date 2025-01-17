package config

var ReverseProxy = ReverseProxyConfig{
	"/service/api":   "http://localhost:3001",
	"/service-1/api": "http://localhost:3002",
}
