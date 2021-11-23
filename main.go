package main

import (
	"embed"
	"fmt"
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

// Version from the Makefile shows what version we're running
var Version string

func days(month time.Time) (days []time.Time) {
	firstDay := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := firstDay.AddDate(0, 1, -1) // add a month, minus a day
	log.WithField("monthEnd", monthEnd).Info("last day")
	for i := 0; i < monthEnd.Day(); i++ {
		days = append(days, firstDay.AddDate(0, 0, i))
	}
	log.WithFields(log.Fields{"month": month, "days": days}).Debug("days of a month")
	return days
}

func main() {
	t, err := template.ParseFS(tmpl, "templates/*.html")
	if err != nil {
		log.WithError(err).Fatal("Failed to parse templates")
	}

	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		chosenDate := time.Now()
		inputMonth := r.URL.Query().Get("month")
		if inputMonth != "" {
			chosenDate, err = time.Parse("2006-01", r.URL.Query().Get("month"))
			if err != nil {
				log.Warn("bad input, defaulting to current month")
				chosenDate = time.Now()
			}
		}
		rw.Header().Set("Content-Type", "text/html")
		err = t.ExecuteTemplate(rw, "index.html", struct {
			Now     time.Time
			Month   time.Time
			Days    []time.Time
			Version string
		}{
			time.Now(),
			chosenDate,
			days(chosenDate),
			Version})
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			log.WithError(err).Fatal("Failed to execute templates")
		}
	})

	log.SetHandler(jsonhandler.Default)

	if _, ok := os.LookupEnv("AWS_LAMBDA_FUNCTION_NAME"); ok {
		err = gateway.ListenAndServe("", nil)
	} else {
		log.SetHandler(text.Default)
		err = http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), nil)
	}
	log.WithError(err).Fatal("error listening")
}
