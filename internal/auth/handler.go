package auth



type Handler struct {
	authService    AuthService
	oauthProviders map[string]OAuthProvider
	frontendURL    string
}

func NewHandler(
	authService AuthService,
	oauthProviders map[string]OAuthProvider,
	frontendURL string,
) *Handler {
	return &Handler{
		authService:    authService,
		oauthProviders: oauthProviders,
		frontendURL:    frontendURL,
	}
}


