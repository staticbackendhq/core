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

	// explicitly set flags for Headless/CI environments
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoSandbox,
		chromedp.DisableGPU,
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Headless,
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
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
