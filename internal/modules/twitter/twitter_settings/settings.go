package twitter_settings

type TwitterSettings struct {
	Api struct {
		UseGraphQlApi bool `mapstructure:"use_graph_ql_api"`
	} `mapstructure:"api"`
	// extracts the twitter ID on the first request instead of just tracking the URL
	// since following URLs will fail whenever the user renames
	ConvertNameToId bool `mapstructure:"convert_name_to_id"`
	// this setting basically allows us to always use the same folder
	// even if the user changes his name (or use any path you'd like)
	UseSubFolderForAuthorName bool `mapstructure:"use_sub_folder_for_author_name"`
	// timeline search endpoint has highly limited rates causing our auth token to get invalidated
	// after a while (didn't figure rate limit out yet), so instead of throwing away all previous requests
	// we can add additional fallbacks to continue our requests (will not work with private/follow only users)
	FallbackAuthTokens []string `mapstructure:"fallback_auth_tokens"`
}
