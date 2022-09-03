package kemono

import (
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func (m *kemono) processDownloadQueue(item *models.TrackedItem, downloadQueue []*postItem) error {
	log.WithField("module", m.Key).Info(
		fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), item.URI),
	)

	for index, data := range downloadQueue {
		log.WithField("module", m.Key).Info(
			fmt.Sprintf(
				"downloading updates for uri: \"%s\" (%0.2f%%)",
				item.URI,
				float64(index+1)/float64(len(downloadQueue))*100,
			),
		)

		if err := m.downloadPost(item, data); err != nil {
			return err
		}

		m.DbIO.UpdateTrackedItem(item, data.id)
	}

	return nil
}

func (m *kemono) downloadPost(item *models.TrackedItem, data *postItem) error {
	response, err := m.Session.Get(data.uri)
	if err != nil {
		return err
	}

	content, _ := io.ReadAll(response.Body)

	for index, link := range m.getDownloadLinks(string(content)) {
		parsedLink, parsedErr := url.Parse(link)
		if parsedErr != nil {
			return parsedErr
		}

		fileName := fp.SanitizePath(parsedLink.Query().Get("f"), false)
		if err = m.Session.DownloadFile(
			path.Join(
				viper.GetString("download.directory"),
				m.Key,
				fp.TruncateMaxLength(m.getSubFolder(item)),
				fp.TruncateMaxLength(fp.SanitizePath(data.id, false)),
				fp.TruncateMaxLength(strings.TrimSpace(fmt.Sprintf("%s_%d_%s", data.id, index+1, fileName))),
			),
			link,
		); err != nil {
			return err
		}
	}

	factory := modules.GetModuleFactory()
	for _, externalURL := range m.getExternalLinks(string(content)) {
		if factory.CanParse(externalURL) {
			module := modules.GetModuleFactory().GetModuleFromURI(externalURL)
			module.InitializeModule()
			newItem := m.DbIO.GetFirstOrCreateTrackedItem(externalURL, "", module)
			if m.Cfg.Run.ForceNew && newItem.CurrentItem != "" {
				log.WithField("module", m.Key).Info(
					fmt.Sprintf("resetting progress for item %s (current id: %s)", newItem.URI, newItem.CurrentItem),
				)
				newItem.CurrentItem = ""
				m.DbIO.ChangeTrackedItemCompleteStatus(newItem, false)
				m.DbIO.UpdateTrackedItem(newItem, "")
			}

			if err = module.Parse(newItem); err != nil {
				return err
			}
		} else {
			log.WithField("module", m.Key).Warnf("unable to parse found URL: \"%s\"", externalURL)
		}
	}

	return nil
}

func (m *kemono) getExternalLinks(html string) (links []string) {
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	document.Find("div.post__body a:not([href^=\"/data\"])").Each(func(index int, row *goquery.Selection) {
		uri, _ := row.Attr("href")
		if fileURL, parseErr := url.Parse(uri); parseErr == nil {
			links = append(links, fileURL.String())
		}
	})

	return links
}

func (m *kemono) getDownloadLinks(html string) (links []string) {
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	document.Find("a[href*='/data/'][href*='?f=']").Each(func(index int, row *goquery.Selection) {
		uri, _ := row.Attr("href")
		if fileURL, parseErr := url.Parse(uri); parseErr == nil {
			alreadyAdded := false
			downloadLink := m.baseUrl.ResolveReference(fileURL).String()

			for _, link := range links {
				if link == downloadLink {
					alreadyAdded = true
					break
				}
			}

			// only add links we didn't add yet
			if !alreadyAdded {
				links = append(links, m.baseUrl.ResolveReference(fileURL).String())
			}
		}
	})

	return links
}
