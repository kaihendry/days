package main

import (
	"embed"
	"html/template"
	"net/http"
	"os"
	"time"

	"github.com/apex/gateway/v2"
	"github.com/apex/log"
	jsonhandler "github.com/apex/log/handlers/json"
	"github.com/apex/log/handlers/text"
)

//go:embed templates
var tmpl embed.FS
var Version string

func days(month time.Time) (days []time.Time) {
	monthEnd := month.AddDate(0, 1, -1) // add a month, minus a day
	for i := 0; i < monthEnd.Day(); i++ {
		days = append(days, month.AddDate(0, 0, i))
	}
	return
}

func main() {
	t, err := template.ParseFS(tmpl, "templates/*.html")
	if err != nil {
		log.WithError(err).Fatal("Failed to parse templates")
	}

	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		chosenDate, err := time.Parse("2006-01", r.URL.Query().Get("month"))
		if err != nil {
			log.Warn("bad input, defaulting to current month")
			chosenDate = time.Now()
		}
		rw.Header().Set("Content-Type", "text/html")
		err = t.ExecuteTemplate(rw, "index.html", struct {
			Month   time.Time
			Days    []time.Time
			Time    time.Time
			Version string
		}{chosenDate, days(chosenDate), time.Now(), Version})
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			log.WithError(err).Fatal("Failed to execute templates")
		}
	})

	port := os.Getenv("_LAMBDA_SERVER_PORT")
	if port == "" {
		log.SetHandler(text.Default)
		err = http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	} else {
		log.SetHandler(jsonhandler.Default)
		err = gateway.ListenAndServe("", nil)
	}
	log.Fatalf("failed to start server: %v", err)

}
