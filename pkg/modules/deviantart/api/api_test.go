package api

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/modules/deviantart/api/internal"
	"testing"
)

func TestNewDeviantartAPI(t *testing.T) {
	daAPI := NewDeviantartAPI("deviantart API")

	fmt.Println(internal.AuthCodeURLImplicit(daAPI.OAuth2Config, "session-id"))
}
