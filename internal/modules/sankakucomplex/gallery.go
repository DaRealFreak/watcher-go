package sankakucomplex

import (
	"fmt"
	"net/url"
	"path"
	"strconv"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	log "github.com/sirupsen/logrus"
)

// parseGallery parses galleries based on the tags in the tracked item
func (m *sankakuComplex) parseGallery(item *models.TrackedItem) (galleryItems []*downloadGalleryItem, err error) {
	originalTag, err := m.extractItemTag(item)
	if err != nil {
		return nil, err
	}

	tag := originalTag
	nextItem := ""
	foundCurrentItem := false

	for !foundCurrentItem {
		apiGalleryResponse, apiErr := m.api.GetPosts(tag, nextItem)
		if apiErr != nil {
			return nil, apiErr
		}

		if nextItem == "" && len(apiGalleryResponse.Data) == 0 {
			log.WithField("module", m.Key).Warn(
				fmt.Sprintf("first request has no results, tag probably changed for uri %s", item.URI),
			)
		}

		nextItem = apiGalleryResponse.Meta.Next

		for _, data := range apiGalleryResponse.Data {
			itemTimestamp := data.CreatedAt.S

			currentItemTimestamp, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
			if item.CurrentItem == "" || itemTimestamp > currentItemTimestamp {
				// direct link with ID is https://www.sankakucomplex.com/post/show/{data.ID}
				// in recent updates they used slugs, so we couldn't directly retrieve the URL based on the ID
				if data.FileURL != "" {
					galleryItems = append(galleryItems, &downloadGalleryItem{
						item: &models.DownloadQueueItem{
							ItemID: strconv.FormatInt(data.CreatedAt.S, 10),
							DownloadTag: path.Join(
								fp.TruncateMaxLength(fp.SanitizePath(m.getDownloadTag(item), false)),
							),
							FileName: fmt.Sprintf(
								"%d_%s_%s",
								data.CreatedAt.S,
								fp.GetFileName(data.ID),
								fp.GetFileName(data.FileURL),
							),
							FileURI:         data.FileURL,
							FallbackFileURI: data.SampleURL,
						},
						apiData: data,
					})
				}
			} else {
				foundCurrentItem = true
				break
			}
		}

		// we reached the last possible page, break here
		if len(apiGalleryResponse.Data) == 0 || nextItem == "" {
			break
		}
	}

	// reverse queue to get the oldest "new" item first and manually update it
	for i, j := 0, len(galleryItems)-1; i < j; i, j = i+1, j-1 {
		galleryItems[i], galleryItems[j] = galleryItems[j], galleryItems[i]
	}

	return galleryItems, nil
}

// extractItemTag extracts the tag from the passed item URL
func (m *sankakuComplex) extractItemTag(item *models.TrackedItem) (string, error) {
	u, _ := url.Parse(item.URI)
	q, _ := url.ParseQuery(u.RawQuery)

	if len(q["tags"]) == 0 {
		return "", fmt.Errorf("parsed uri(%s) does not contain any \"tags\" tag", item.URI)
	}

	return q["tags"][0], nil
}

func (m *sankakuComplex) getDownloadTag(item *models.TrackedItem) string {
	if item.SubFolder != "" {
		return item.SubFolder
	}

	originalTag, err := m.extractItemTag(item)
	if err != nil {
		return ""
	}

	return originalTag
}
