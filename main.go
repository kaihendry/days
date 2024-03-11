package main

import (
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/apex/gateway/v2"
)

//go:embed templates
var tmpl embed.FS

type day struct {
	Date      time.Time
	IsHoliday bool
}

func days(month time.Time) (days []day) {
	firstDay := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := firstDay.AddDate(0, 1, -1) // add a month, minus a day
	slog.Info("last day", "monthEnd", monthEnd)
	for i := 0; i < monthEnd.Day(); i++ {
		days = append(days, day{Date: firstDay.AddDate(0, 0, i)})
	}
	slog.Debug("days of a month", "month", month, "days", days)
	return days
}

func getWeekNumber(t time.Time) int {
	_, week := t.ISOWeek()
	return week
}

func main() {

	commit, _ := GitCommit()

	t, err := template.New("base").Funcs(template.FuncMap{"weekNumber": getWeekNumber}).ParseFS(tmpl, "templates/*.html")
	if err != nil {
		slog.Error("Failed to parse templates", "error", err)
		return
	}

	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		chosenDate := time.Now()
		inputMonth := r.URL.Query().Get("month")
		if inputMonth != "" {
			chosenDate, err = time.Parse("2006-01", r.URL.Query().Get("month"))
			if err != nil {
				chosenDate = time.Now()
				slog.Warn("bad input, defaulting to current month", "month", chosenDate.Format("2006-01"))
			}
		}
		rw.Header().Set("Content-Type", "text/html")
		err = t.ExecuteTemplate(rw, "index.html", struct {
			Now     time.Time
			Month   time.Time
			Days    []day
			Version string
		}{
			time.Now(),
			chosenDate,
			days(chosenDate),
			commit})
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			slog.Error("Failed to execute templates", "error", err)
		}
	})

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if _, ok := os.LookupEnv("AWS_LAMBDA_FUNCTION_NAME"); ok {
		err = gateway.ListenAndServe("", nil)
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
		err = http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), nil)
	}
	slog.Error("error listening", "error", err)
}

func GitCommit() (commit string, dirty bool) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false
	}
	for _, setting := range bi.Settings {
		switch setting.Key {
		case "vcs.modified":
			dirty = setting.Value == "true"
		case "vcs.revision":
			commit = setting.Value
		}
	}
	return
}
