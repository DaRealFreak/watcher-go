package x_transaction_id

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/modules/twitter/twitter_settings"
	"github.com/PuerkitoBio/goquery"
	"io"
	"math"
	"math/rand"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	additionalRandomNumber = 3
	defaultKeyword         = "obfiowerehiring"
	// Twitter's custom epoch: Apr 30, 2023 00:00:00 UTC in seconds
	twitterEpochSec = 1682924400
)

type XTransactionIdHandler struct {
	transactionSession     http.TlsClientSessionInterface
	settings               twitter_settings.TwitterSettings
	onDemandFileRegex      *regexp.Regexp
	indicesRegex           *regexp.Regexp
	defaultRowIndex        int
	defaultKeyBytesIndices []int
	key                    string
	keyBytes               []byte
	animationKey           string
}

func NewXTransactionIdHandler(transactionSession http.TlsClientSessionInterface, settings twitter_settings.TwitterSettings) *XTransactionIdHandler {
	handler := &XTransactionIdHandler{
		transactionSession: transactionSession,
		settings:           settings,
		// Matches "'ondemand.s':'<token>'" in the HTML
		onDemandFileRegex: regexp.MustCompile(`['"]ondemand\.s['"]:\s*['"]([\w]+)['"]`),
		// Matches "(x[NN], 16)" in the JS file
		indicesRegex: regexp.MustCompile(`\(\w\[(\d{1,2})\],\s*16\)`),
	}
	return handler
}

// GenerateTransactionId runs the migration flow, pulls out the indices,
// then extracts and decodes the Twitter-site-verification key.
func (h *XTransactionIdHandler) GenerateTransactionId(
	method, path string,
	responseDoc *goquery.Document,
	keyOverride, animKeyOverride string,
	timeNowOverride int64,
	xOrPrefix uint8,
) (string, error) {
	// 1) Figure out timestamp (in seconds since Twitter epoch)
	var tsSec int64
	if timeNowOverride > 0 {
		tsSec = timeNowOverride
	} else {
		nowMs := time.Now().UnixNano() / int64(time.Millisecond)
		tsSec = (nowMs - twitterEpochSec*1000) / 1000
	}
	// pack into 4 little‐endian bytes
	ts := int(tsSec)
	timeBytes := []byte{
		byte(ts),
		byte(ts >> 8),
		byte(ts >> 16),
		byte(ts >> 24),
	}

	// 2) If we don't have a home page document, we need to scrape it if both keys are empty
	if responseDoc == nil && (h.key == "" || h.animationKey == "") {
		err := h.ExtractAnimationKey()
		if err != nil {
			return "", err
		}
	}

	// 2) Determine key string
	key := keyOverride
	if key == "" {
		if h.key != "" {
			key = h.key
		} else {
			// need to scrape it
			k, err := h.getKey(responseDoc)
			if err != nil {
				return "", err
			}
			key = k
		}
	}
	h.key = key

	// decode into bytes
	keyBytes, err := h.getKeyBytes(key)
	if err != nil {
		return "", err
	}
	h.keyBytes = keyBytes

	// 3) Determine animation key
	animKey := animKeyOverride
	if animKey == "" {
		if h.animationKey != "" {
			animKey = h.animationKey
		} else {
			ak, err := h.getAnimationKey(responseDoc)
			if err != nil {
				return "", err
			}
			animKey = ak
		}
	}
	h.animationKey = animKey

	// 4) SHA-256 on "METHOD!PATH!TIMESTAMPKEYWORDANIMKEY"
	toHash := fmt.Sprintf("%s!%s!%d%s%s", method, path, ts, defaultKeyword, animKey)
	sum := sha256.Sum256([]byte(toHash))

	// 5) Build payload: keyBytes + timeBytes + first16(sum) + additionalRandomNumber
	payload := append([]byte{}, keyBytes...)
	payload = append(payload, timeBytes...)
	payload = append(payload, sum[:16]...)
	payload = append(payload, additionalRandomNumber)

	// 6) XOR‐prefix with a random byte
	if xOrPrefix == 0 {
		xOrPrefix = uint8(rand.Intn(256))
	}
	randByte := xOrPrefix
	out := make([]byte, len(payload)+1)
	out[0] = randByte
	for i, b := range payload {
		out[i+1] = b ^ randByte
	}

	// 7) Base64 encode without padding
	encoded := base64.StdEncoding.EncodeToString(out)
	finalString := strings.TrimRight(encoded, "=")

	return finalString, nil
}

// ExtractAnimationKey extracts the animation key from the current session
func (h *XTransactionIdHandler) ExtractAnimationKey() error {
	// 1) Fetch & parse the page (migration + form)
	doc, err := h.handleMigration()
	if err != nil {
		return err
	}

	// 2) Pull out indices
	rowIdx, byteIdxs, err := h.getIndices(doc)
	if err != nil {
		return err
	}
	h.defaultRowIndex = rowIdx
	h.defaultKeyBytesIndices = byteIdxs

	// 3) Scrape & decode the verification key
	key, err := h.getKey(doc)
	if err != nil {
		return err
	}
	h.key = key

	keyBytes, err := h.getKeyBytes(key)
	if err != nil {
		return err
	}
	h.keyBytes = keyBytes

	// 4) Compute the animation key
	animKey, err := h.getAnimationKey(doc)
	if err != nil {
		return err
	}
	h.animationKey = animKey
	return nil
}

// GetAnimationKey computes what your Python called get_animation_key.
// It assumes h.homeDoc and h.defaultRowIndex / h.defaultKeyBytesIndices / h.keyBytes
// are already populated (i.e. after you call GetTransactionId up through decoding keyBytes).
func (h *XTransactionIdHandler) getAnimationKey(doc *goquery.Document) (string, error) {
	const totalTime = 4096.0

	// 1) row index is keyBytes[DEFAULT_ROW_INDEX] % 16
	rowIndex := int(h.keyBytes[h.defaultRowIndex]) % 16

	// 2) frameTime is the product of each keyBytes[idx] % 16
	frameTime := 1
	for _, idx := range h.defaultKeyBytesIndices {
		frameTime *= int(h.keyBytes[idx]) % 16
	}

	// 3) round frameTime to the nearest 10 (just like Python’s round(.../10)*10)
	frameTime = int(math.Round(float64(frameTime)/10.0) * 10)

	// 3) grab the 2D array of ints from the SVG path data
	arr, err := h.get2DArray(doc)
	if err != nil {
		return "", fmt.Errorf("build 2D array: %w", err)
	}
	if rowIndex >= len(arr) {
		return "", fmt.Errorf("rowIndex %d out of range", rowIndex)
	}
	frameRow := arr[rowIndex]

	// 4) compute normalized targetTime
	targetTime := float64(frameTime) / totalTime

	// 5) call your animate (must return the hex‐string key)
	animationKey, err := h.animate(frameRow, targetTime)
	if err != nil {
		return "", fmt.Errorf("animate: %w", err)
	}
	return animationKey, nil
}

// get2DArray replicates your Python get_2d_array:
//   - finds the SVG path in #loading-x-anim layers
//   - extracts the "d" attribute, strips the leading "M ...", splits on "C"
//   - pulls out numbers into [][]int
func (h *XTransactionIdHandler) get2DArray(doc *goquery.Document) ([][]int, error) {
	frames := doc.Find(`[id^="loading-x-anim"]`)
	if frames.Length() == 0 {
		return nil, fmt.Errorf("no loading-x-anim frames found")
	}

	// choose the 5th byte mod 4
	idx := int(h.keyBytes[5]) % 4
	sel := frames.Eq(idx).Children().Eq(0).Children().Eq(1)
	dAttr, ok := sel.Attr("d")
	if !ok {
		return nil, fmt.Errorf("path has no 'd' attribute")
	}

	// strip off the first 9 chars, split on "C"
	segments := strings.Split(dAttr[9:], "C")
	out := make([][]int, len(segments))

	// regex to isolate runs of digits
	numRe := regexp.MustCompile(`[^\d]+`)
	for i, seg := range segments {
		fields := numRe.Split(strings.TrimSpace(seg), -1)
		row := make([]int, 0, len(fields))
		for _, f := range fields {
			if f == "" {
				continue
			}
			n, err := strconv.Atoi(f)
			if err != nil {
				return nil, fmt.Errorf("invalid number %q: %w", f, err)
			}
			row = append(row, n)
		}
		out[i] = row
	}
	return out, nil
}

// animate port of Python's animate: color+rotation interpolation -> hex key.
func (h *XTransactionIdHandler) animate(frames []int, t float64) (string, error) {
	// 1) Build RGBA “from” and “to” colors
	from := []float64{
		float64(frames[0]),
		float64(frames[1]),
		float64(frames[2]),
		1.0,
	}
	to := []float64{
		float64(frames[3]),
		float64(frames[4]),
		float64(frames[5]),
		1.0,
	}

	// 2) Compute the end rotation (deg)
	rotDeg := solve(float64(frames[6]), 60.0, 360.0, true)

	// 3) Build cubic-bezier control points from frames[7:]
	curves := make([]float64, len(frames)-7)
	for i, v := range frames[7:] {
		// Python uses minVal = -1 for odd, 0 for even
		var minVal float64
		if i%2 == 1 {
			minVal = -1.0
		} else {
			minVal = 0.0
		}
		curves[i] = solve(float64(v), minVal, 1.0, false)
	}
	cubic := NewCubic(curves)

	// 4) Sample the curve at t to get eased progress
	val := cubic.GetValue(t)

	// 5) Interpolate colors
	color, err := Interpolate(from, to, val)
	if err != nil {
		return "", err
	}
	// clamp negatives
	for i := range color {
		if color[i] < 0 {
			color[i] = 0
		}
	}

	// 6) Interpolate rotation (we treat it as a 1-element slice)
	rotInterp, err := Interpolate([]float64{0.0}, []float64{rotDeg}, val)
	if err != nil {
		return "", err
	}
	// build the 2×2 rotation matrix
	matrix := ConvertRotationToMatrix(rotInterp[0])

	// 7) Convert the first 3 color channels to hex digits (lower‐case, no padding)
	parts := make([]string, 0, 3+len(matrix)+2)
	for _, c := range color[:3] {
		// round to the nearest int and format as hex (lowercase)
		parts = append(parts, fmt.Sprintf("%x", int(math.Round(c))))
	}

	// 8) Convert each matrix entry to hex via FloatToHex
	for _, m := range matrix {
		// round to two decimals, take abs
		r := math.Round(m*100) / 100
		if r < 0 {
			r = -r
		}
		hexVal := FloatToHex(r) // e.g. "0.7AE147AE147AE" or "1"
		hexVal = strings.ToLower(hexVal)

		if hexVal == "" {
			// Python would emit "" → "0"
			parts = append(parts, "0")
		} else if strings.HasPrefix(hexVal, ".") {
			// Python prefixes fractional with a leading "0"
			parts = append(parts, "0"+hexVal)
		} else {
			parts = append(parts, hexVal)
		}
	}

	// 9) Append the two trailing zeros
	parts = append(parts, "0", "0")

	// 10) Join and strip out dots and hyphens
	joined := strings.Join(parts, "")
	cleaned := regexp.MustCompile(`[.-]`).ReplaceAllString(joined, "")
	return cleaned, nil
}

// getKey finds <meta name="twitter-site-verification" content="…">
func (h *XTransactionIdHandler) getKey(doc *goquery.Document) (string, error) {
	sel := doc.Find(`[name="twitter-site-verification"]`)
	if sel.Length() == 0 {
		return "", fmt.Errorf("couldn't get key from the page source")
	}
	content, exists := sel.Attr("content")
	if !exists || content == "" {
		return "", fmt.Errorf("twitter-site-verification meta had no content")
	}
	return content, nil
}

// getKeyBytes base64-decodes the verification key string.
func (h *XTransactionIdHandler) getKeyBytes(key string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("error decoding key bytes: %w", err)
	}
	return data, nil
}

// handleMigration fetches the home page and looks for a migration URL.
// If found, it submits the form and returns the resulting document; else the document of the home page is returned.
func (h *XTransactionIdHandler) handleMigration() (*goquery.Document, error) {
	resp, err := h.get("https://x.com/")
	if err != nil {
		return nil, err
	}

	doc := h.transactionSession.GetDocument(resp)

	// look for a meta-refresh migration URL or any migrate?tok=… link in the HTML
	migRe := regexp.MustCompile(`(https?://(?:www\.)?(?:twitter|x)\.com(?:/x)?/migrate(?:[/?]\S*?tok=[A-Za-z0-9%\-_]+))`)
	var migrationUrl string

	// a) meta[http-equiv=refresh]
	if content, exists := doc.Find("meta[http-equiv='refresh']").Attr("content"); exists {
		if m := migRe.FindStringSubmatch(content); len(m) > 1 {
			migrationUrl = m[1]
		}
	}
	// b) fallback: search full HTML
	if migrationUrl == "" {
		html, _ := doc.Html()
		if m := migRe.FindStringSubmatch(html); len(m) > 1 {
			migrationUrl = m[1]
		}
	}

	// if we found a migration redirect, follow it
	if migrationUrl != "" {
		resp, err = h.get(migrationUrl)
		if err != nil {
			return nil, err
		}

		doc, err = goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			return nil, err
		}
	}

	form := doc.Find("form[name='f']")
	if form.Length() == 0 {
		form = doc.Find("form[action='https://x.com/x/migrate']")
	}

	// if there's a form, build and submit it
	if form.Length() > 0 {
		action, _ := form.Attr("action")
		if action == "" {
			action = "https://x.com/x/migrate"
		}
		// append the mx=2 query
		action = strings.TrimRight(action, "/") + "/?mx=2"

		method, _ := form.Attr("method")
		method = strings.ToUpper(method)
		if method == "" {
			method = "POST"
		}

		// collect all <input> name/value pairs
		data := url.Values{}
		form.Find("input").Each(func(i int, s *goquery.Selection) {
			if name, ok := s.Attr("name"); ok {
				val, _ := s.Attr("value")
				data.Add(name, val)
			}
		})

		// choose GET vs POST
		if method == "POST" {
			resp, err = h.post(action, data)
		} else {
			resp, err = h.get(action + "?" + data.Encode())
		}

		if err != nil {
			return nil, err
		}

		doc, err = goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			return nil, err
		}
	}

	return doc, nil
}

// getIndices scrapes the ondemand token from the home page doc,
// fetches the corresponding JS, and returns the first index (row)
// and the rest as key-byte indices.
func (h *XTransactionIdHandler) getIndices(homeDoc *goquery.Document) (int, []int, error) {
	// serialize doc to HTML
	html, err := homeDoc.Html()
	if err != nil {
		return 0, nil, fmt.Errorf("couldn't serialize home page: %w", err)
	}

	// extract token
	m := h.onDemandFileRegex.FindStringSubmatch(html)
	if len(m) < 2 {
		return 0, nil, fmt.Errorf("couldn't find ondemand token")
	}
	token := m[1]

	// fetch the JS
	jsURL := fmt.Sprintf("https://abs.twimg.com/responsive-web/client-web/ondemand.s.%sa.js", token)
	resp, err := h.get(jsURL)
	if err != nil {
		return 0, nil, fmt.Errorf("error fetching ondemand file: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("error reading ondemand response: %w", err)
	}
	jsText := string(body)

	// find all "(x[NN], 16)" matches
	matches := h.indicesRegex.FindAllStringSubmatch(jsText, -1)
	if len(matches) == 0 {
		return 0, nil, fmt.Errorf("couldn't get KEY_BYTE indices")
	}

	// convert to ints
	keyByteIndices := make([]int, len(matches))
	for i, sub := range matches {
		idx, convErr := strconv.Atoi(sub[1])
		if convErr != nil {
			return 0, nil, fmt.Errorf("invalid index %q: %w", sub[1], convErr)
		}
		keyByteIndices[i] = idx
	}

	if len(keyByteIndices) < 2 {
		return 0, nil, fmt.Errorf("not enough key byte indices")
	}

	// the first element is rowIndex, the rest are keyByteIndices
	return keyByteIndices[0], keyByteIndices[1:], nil
}
