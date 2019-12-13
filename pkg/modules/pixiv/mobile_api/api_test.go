package mobile_api

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/url"
	"os"
	"testing"
)

func getTestMobileAPI() *MobileAPI {
	testAccount := models.Account{
		Username: os.Getenv("PIXIV_USER"),
		Password: os.Getenv("PIXIV_PASS"),
	}

	mobileAPI, _ := NewMobileAPI("pixiv Mobile API", testAccount)

	return mobileAPI
}

func TestNewMobileAPI(t *testing.T) {
	mobileAPI := getTestMobileAPI()

	apiURL, _ := url.Parse("https://app-api.pixiv.net/v1/illust/detail")
	data := url.Values{
		"illust_id": {"123456"},
	}
	apiURL.RawQuery = data.Encode()

	res, err := mobileAPI.Session.Get(apiURL.String())
	if err != nil {
		panic(err)
	}

	fmt.Println(ioutil.ReadAll(res.Body))
}

func TestLogin(t *testing.T) {
	testAccount := models.Account{
		Username: os.Getenv("PIXIV_USER"),
		Password: os.Getenv("PIXIV_PASS"),
	}

	mobileAPI, err := NewMobileAPI("pixiv Mobile API", testAccount)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(mobileAPI)
}
