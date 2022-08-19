package staticbackend

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/staticbackendhq/core/logger"
)

var (
	views map[string]*template.Template
)

func loadTemplates() error {
	var partials []string
	entries, err := os.ReadDir("./templates/partials")
	if err != nil {
		return err
	}

	funcs := customFuncs()

	for _, e := range entries {
		partials = append(partials, fmt.Sprintf("./templates/partials/%s", e.Name()))
	}

	views = make(map[string]*template.Template)

	tmpls, err := os.ReadDir("./templates")
	if err != nil {
		return err
	}

	for _, tmpl := range tmpls {
		name := fmt.Sprintf("./templates/%s", tmpl.Name())
		if !strings.HasSuffix(name, ".html") {
			continue
		}

		cur := append([]string{name}, partials...)

		t, err := template.New(tmpl.Name()).Funcs(funcs).ParseFiles(cur...)
		if err != nil {
			return err
		}

		views[tmpl.Name()] = t
	}

	return nil
}

type Flash struct {
	Type    string
	Message string
}

type ViewData struct {
	Title      string
	Language   string
	ActiveMenu string
	Flash      *Flash
	Data       interface{}
}

func render(w http.ResponseWriter, r *http.Request, view string, data interface{}, flash *Flash, log *logger.Logger) {
	renderWithMenu(w, r, view, data, flash, "", log)
}

func renderWithMenu(w http.ResponseWriter, r *http.Request, view string, data interface{}, flash *Flash, menu string, log *logger.Logger) {
	vd := ViewData{
		ActiveMenu: menu,
		Data:       data,
		Flash:      flash,
	}

	tmpl, ok := views[view]
	if !ok {
		http.Error(w, fmt.Sprintf(`template "%s" cannot be found`, view), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, vd); err != nil {
		log.Error().Err(err).Msgf(`error executing template "%s"`, view)

		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func renderErr(w http.ResponseWriter, r *http.Request, err error, log *logger.Logger) {
	if err != nil {
		log.Error().Err(err).Stack().Msg("err in ui")
	}

	render(w, r, "err.html", nil, nil, log)
}

func customFuncs() template.FuncMap {
	return template.FuncMap{
		"getField": func(s string, doc map[string]interface{}) string {
			v, ok := doc[s]
			if !ok {
				return "n/a"
			}

			date, ok := doc[s].(time.Time)
			if ok {
				return date.Format("2006-01-02 15:04")
			}

			if s == "sb_posted" {
				i, err := strconv.ParseInt(fmt.Sprintf("%v", doc[s]), 10, 64)
				if err == nil {
					ts := time.Unix(i/1000, 0)
					return ts.Format("2006-01-02 15:04")
				}
			}
			return fmt.Sprintf("%v", v)
		},
		"convertFileSize": func(size int64) string {
			const unit = 1000
			if size < unit {
				return fmt.Sprintf("%d B", size)
			}
			div, exp := int64(unit), 0
			for n := size / unit; n >= unit; n /= unit {
				div *= unit
				exp++
			}

			return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
		},
		"convertFileType": func(key string) string {
			return filepath.Ext(key)[1:]
		},
		"convertFileUploadedDate": func(uploaded time.Time) string {
			return uploaded.Format("January 02, 2006 at 15:04")
		},
		"parseFilename": func(key string) string {
			splitedKey := strings.Split(key, "/")

			filename := splitedKey[len(splitedKey)-1]

			return strings.Split(filename, ".")[0]
		},
		"getElementByFileExt": func(fileType string) string {
			imgTypes := "jpg png gif jpeg svg icon webp raw"
			videoTypes := "avi mp4 mkv mov wmv flv avchd"
			audioTypes := "mp3 ogg aac oga flac pcm wav aiff"

			fileType = strings.ToLower(fileType)

			if strings.Contains(imgTypes, fileType) {
				return "image"
			} else if strings.Contains(videoTypes, fileType) {
				return "video"
			} else if strings.Contains(audioTypes, fileType) {
				return "audio"
			}

			return ""
		},
	}
}
