package main

import (
	"context"
	"encoding/json"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

type ConvertParams struct {
	ToPDF    bool   `json:"toPDF"`
	URL      string `json:"url"`
	FullPage bool   `json:"fullpage"`
}

func Do(body []byte) (buf []byte, err error) {
	var data ConvertParams
	if err = json.Unmarshal(body, &data); err != nil {
		return
	}

	/*opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("disable-gpu", true),
	)*/

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	err = chromedp.Run(ctx, toBytes(data, &buf))
	return buf, err
}

func toBytes(data ConvertParams, res *[]byte) chromedp.Tasks {
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
