package main

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/apex/gateway/v2"
	ics "github.com/arran4/golang-ical"
)

//go:embed templates
var tmpl embed.FS

type day struct {
	Date           time.Time
	IsHoliday      bool
	HolidaySummary string
}

func days(month time.Time) (days []day) {
	firstDay := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := firstDay.AddDate(0, 1, -1) // add a month, minus a day
	slog.Debug("last day", "monthEnd", monthEnd)
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
		days := days(chosenDate)

		icsURL := r.URL.Query().Get("ics")
		// check icsURL is indeed a URL
		if icsURL != "" {
			// get ics file
			icsData, err := fetch(icsURL)
			if err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				slog.Error("Failed to fetch ics", "error", err)
				return
			}
			cal, err := ics.ParseCalendar(strings.NewReader(icsData))
			if err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				slog.Error("Failed to parse ics", "error", err)
				return
			}
			holidays, err := findHolidays(cal)
			if err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				slog.Error("Failed to find holidays", "error", err)
				return
			}
			for i, d := range days {
				for _, h := range holidays {
					start, err := h.GetDtStampTime()
					if err != nil {
						http.Error(rw, err.Error(), http.StatusInternalServerError)
						slog.Error("Failed to get start time", "error", err)
						return
					}
					end, err := h.GetEndAt()
					if err != nil {
						http.Error(rw, err.Error(), http.StatusInternalServerError)
						slog.Error("Failed to get end time", "error", err)
						return
					}
					if (d.Date.After(start) || d.Date.Equal(start)) && (d.Date.Before(end) || d.Date.Equal(end)) {
						days[i].IsHoliday = true
						days[i].HolidaySummary = h.GetProperty(ics.ComponentPropertySummary).Value
					}
				}
			}
		}

		rw.Header().Set("Content-Type", "text/html")
		err = t.ExecuteTemplate(rw, "index.html", struct {
			Now      time.Time
			Month    time.Time
			Previous time.Time
			Next     time.Time
			Days     []day
			Version  string
			IcsURL   string
		}{
			time.Now(),
			chosenDate,
			chosenDate.AddDate(0, -1, 0),
			chosenDate.AddDate(0, 1, 0),
			days,
			commit,
			icsURL})
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

func findHolidays(cal *ics.Calendar) (holidays []ics.VEvent, err error) {
	for _, event := range cal.Events() {
		start, err := event.GetDtStampTime()
		if err != nil {
			return nil, err
		}
		end, err := event.GetEndAt()
		if err != nil {
			return nil, err
		}
		// if duration is above a day, it's a holiday
		if end.Sub(start).Hours()/24 > 1 {
			holidays = append(holidays, *event)
		}
	}
	return holidays, nil
}

func fetch(url string) (string, error) {
	c := http.Client{Timeout: 1 * time.Second}

	slog.Debug("http get", "url", url)
	resp, err := c.Get(url)
	if err != nil {
		return "", err
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			slog.Error("failed to close response body", "error", err)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download %s: %s", url, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
