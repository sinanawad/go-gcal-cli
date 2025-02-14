/**
 * @license
 * Copyright Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
// [START calendar_quickstart]
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

type model struct {
	events []*calendar.Event
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
	}
	return m, nil
}

const (
	// ClientSecretPath is the path to the client secret file.
	startedMeeting = "+"
	nextMeeting    = ">"
)

func prepareTableRows(events calendar.Events) [][]string {

	var rows [][]string
	var timeNow = time.Now()
	for _, item := range events.Items {
		date := item.Start.DateTime
		if date == "" { // remove all day events
			continue
		}
		startTime, _ := time.Parse(time.RFC3339, item.Start.DateTime)
		endTime, _ := time.Parse(time.RFC3339, item.End.DateTime)

		if startTime == endTime {
			continue
		}

		if timeNow.After(endTime) {
			continue
		}

		if endTime.Sub(startTime) > 24*time.Hour {
			continue
		}

		if timeNow.After(startTime) && timeNow.Before(endTime) {
			item.Summary = startedMeeting + item.Summary
		} else if startTime.Sub(timeNow) < 10*time.Minute {
			item.Summary = nextMeeting + item.Summary
		}

		if len(item.Summary) > 57 {
			item.Summary = item.Summary[:57] + "..."
		}

		rows = append(rows, []string{item.Summary, startTime.Format("15:04"), endTime.Format("15:04"), item.HangoutLink})
		if len(rows) > 5 {
			return rows
		}
	}
	return rows

}

func (m model) View() string {
	var output string

	header := lipgloss.NewStyle().Align(lipgloss.Center).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("0")).Render
	oldStyle := lipgloss.NewStyle().Align(lipgloss.Center).Foreground(lipgloss.Color("9")).Background(lipgloss.Color("0")).Render
	newStyle := lipgloss.NewStyle().Align(lipgloss.Center).Foreground(lipgloss.Color("10")).Background(lipgloss.Color("0")).Render
	currentStyle := lipgloss.NewStyle().Align(lipgloss.Center).Foreground(lipgloss.Color("2")).Background(lipgloss.Color("0")).Render

	output += header(fmt.Sprintf("%-50s %-5s-%-5s %-20s\n", "Summary", "Start", "End", "Hangout Link"))

	for i, event := range m.events {
		startTime, _ := time.Parse(time.RFC3339, event.Start.DateTime)
		endTime, _ := time.Parse(time.RFC3339, event.End.DateTime)
		now := time.Now()

		if event.Start.DateTime == "" {
			continue
		}

		style := oldStyle
		if startTime.Before(now) {
			style = oldStyle
		} else if endTime.Before(now) {
			style = newStyle
		} else {
			style = currentStyle
		}

		if len(event.Summary) > 47 {
			event.Summary = event.Summary[:47] + "..."
		}
		output += style(fmt.Sprintf("%-50s %-5s-%-5s %-20s\n", event.Summary, startTime.Format("15:04"), endTime.Format("15:04"), event.HangoutLink))

		//		output += style.Render(fmt.Sprintf("%-30s %-20s %-20s %-50s\n", event.Summary, startTime.Format("15:04"), endTime.Format("15:04"), event.HangoutLink))
		if i == 10 {
			break
		}
	}

	return output
}

func runBubbleTea(events []*calendar.Event) {
	p := tea.NewProgram(model{events: events})
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
	}
}

var (
	HeaderStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("0"))
	NormalStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Background(lipgloss.Color("0"))
	StartedRowStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#62D9F5")).Background(lipgloss.Color("#0000FF"))
	NextRowStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#000000")).Background(lipgloss.Color("#00FF00"))
)

func main() {
	ctx := context.Background()
	b, err := os.ReadFile("go-gcal-cli-credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, calendar.CalendarReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Calendar client: %v", err)
	}

	t := time.Now().AddDate(0, 0, -1).Format(time.RFC3339)

	tMax := time.Now().AddDate(0, 0, 1).Format(time.RFC3339)
	//events, err := srv.Events.List("primary").ShowDeleted(false).SingleEvents(true).TimeMin(t).TimeMax(tMax).OrderBy("startTime").Do()
	events, err := srv.Events.List("primary").ShowDeleted(false).SingleEvents(true).TimeMin(t).TimeMax(tMax).OrderBy("startTime").Do()

	if err != nil {
		log.Fatalf("Unable to retrieve next ten of the user's events: %v", err)
	}

	//style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9")).Background(lipgloss.Color("0")).Render

	if len(events.Items) == 0 {
		fmt.Println("No upcoming events found.")
	} else {
		for _, item := range events.Items {
			date := item.Start.DateTime
			if date == "" {
				continue
				//date = item.Start.Date
			}
			//fmt.Printf("%v (%v) %v\n", item.Summary, date, item.HangoutLink)
			startTime, _ := time.Parse(time.RFC3339, item.Start.DateTime)
			endTime, _ := time.Parse(time.RFC3339, item.End.DateTime)

			if startTime == endTime {
				continue
			}

			if len(item.Summary) > 47 {
				item.Summary = item.Summary[:47] + "..."
			}

			//		fmt.Printf("%-50s %-5s-%-5s %-20s\n", item.Summary, startTime.Format("15:04"), endTime.Format("15:04"), item.HangoutLink)
			//fmt.Printf(style(fmt.Sprintf("%v\t%v\t%v\t%v\n", item.Summary, startTime.Format("15:04"), endTime.Format("15:04"), item.HangoutLink)))
		}
	}

	rows := prepareTableRows(*events)

	tbl := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("99"))).
		StyleFunc(func(row, col int) lipgloss.Style {

			if row == -1 {
				return HeaderStyle
			}

			if row > -1 {
				if len(rows[row][0]) > 0 && rows[row][0][0] == nextMeeting[0] {
					return NextRowStyle
				}

				if len(rows[row][0]) > 0 && rows[row][0][0] == startedMeeting[0] {
					return StartedRowStyle
				}

			}

			return NormalStyle
		}).
		Headers("Summary", time.Now().Format("15:04"), "End", "Link").
		Rows(rows...)

	fmt.Println(tbl.Render())
	//runBubbleTea(events.Items)

}
