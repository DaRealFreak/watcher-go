package api

import (
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"os"
	"testing"
)

func TestNewDeviantartAPI(t *testing.T) {
	testAccount := &models.Account{
		Username: os.Getenv("DEVIANTART_USER"),
		Password: os.Getenv("DEVIANTART_PASS"),
	}

	daAPI := NewDeviantartAPI("deviantart API", testAccount)
	daAPI.AddRoundTrippers()
}
