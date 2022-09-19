package staticbackend

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/extra"
	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/logger"
	"github.com/staticbackendhq/core/middleware"
	"github.com/staticbackendhq/core/model"
	"github.com/staticbackendhq/core/sms"
)

type extras struct {
	log *logger.Logger
}

func (ex *extras) resizeImage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		ex.log.Error().Err(err).Msg("cannot parse form")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	config, auth, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file, h, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// check for file size
	// there's a maximum of 2GB for image manupulation
	if h.Size/(1000*1000) > 2 {
		http.Error(w, "file size exeeded your limit", http.StatusBadRequest)
		return
	}

	ext := filepath.Ext(h.Filename)

	//TODO: Remove all but a-zA-Z/ from name

	name := r.Form.Get("name")
	if len(name) == 0 {
		name = internal.RandStringRunes(32)
	}

	newWidth, err := strconv.ParseFloat(r.Form.Get("width"), 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var b []byte
	buf := bytes.NewBuffer(b)

	if err := extra.ResizeImage(h.Filename, file, buf, newWidth); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fileKey := fmt.Sprintf("%s/%s/%s%s",
		config.Name,
		auth.AccountID,
		name,
		ext,
	)

	resizedBytes := buf.Bytes()

	ex.log.Info().Msgf("resized bytes: %d", len(resizedBytes))
	upData := model.UploadFileData{FileKey: fileKey, File: bytes.NewReader(resizedBytes)}
	url, err := backend.Filestore.Save(upData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	f := model.File{
		AccountID: auth.AccountID,
		Key:       fileKey,
		URL:       url,
		Size:      int64(len(b)),
		Uploaded:  time.Now(),
	}

	newID, err := backend.DB.AddFile(config.Name, f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := new(struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	})
	data.ID = newID
	data.URL = url

	respond(w, http.StatusOK, data)
}

func (ex *extras) sudoSendSMS(w http.ResponseWriter, r *http.Request) {
	var data sms.SMSData
	if err := parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := sms.Send(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}

type ConvertParam struct {
	ToPDF    bool   `json:"toPDF"`
	URL      string `json:"url"`
	FullPage bool   `json:"fullpage"`
}

func (ex *extras) htmlToX(w http.ResponseWriter, r *http.Request) {
	config, auth, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var data ConvertParam
	if parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	/*opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("disable-gpu", true),
	)*/

	// make sure it can timeout
	cctx, _ := context.WithTimeout(context.Background(), 15*time.Second)

	//HACK:
	ctx, cancel := chromedp.NewContext(cctx)
	//ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var buf []byte

	if err := chromedp.Run(ctx, ex.toBytes(data, &buf)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ext := ".png"
	if data.ToPDF {
		ext = ".pdf"
	}

	fileKey := fmt.Sprintf("%s/%s/%d%s",
		config.Name,
		auth.AccountID,
		time.Now().UnixNano(),
		ext,
	)

	ufd := model.UploadFileData{
		FileKey: fileKey,
		File:    bytes.NewReader(buf),
	}
	fileURL, err := backend.Filestore.Save(ufd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	f := model.File{
		AccountID: auth.AccountID,
		Key:       fileKey,
		URL:       fileURL,
		Size:      int64(len(buf)),
		Uploaded:  time.Now(),
	}

	newID, err := backend.DB.AddFile(config.Name, f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result := new(struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	})
	result.ID = newID
	result.URL = fileURL

	respond(w, http.StatusOK, data)
}

func (ex *extras) toBytes(data ConvertParam, res *[]byte) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.EmulateViewport(1280, 768),
		chromedp.Navigate(data.URL),
		chromedp.WaitReady("body"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var buf []byte
			var err error
			if data.ToPDF {
				buf, _, err = page.PrintToPDF().Do(ctx)
			} else {
				params := page.CaptureScreenshot()
				// TODO: This should capture full screen ?!?
				params.CaptureBeyondViewport = data.FullPage

				buf, err = params.Do(ctx)
			}
			if err != nil {
				return err
			}

			*res = buf
			return nil

		}),
	}
}
