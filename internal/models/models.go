package models

type User struct {
	AccountID    string `json:"accountId"`
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
	Active       bool   `json:"active"`
}

type Issue struct {
	Key    string      `json:"key"`
	Fields IssueFields `json:"fields"`
}

type IssueFields struct {
	Summary     string       `json:"summary"`
	Description *ADFDocument `json:"description"`
	Status      Status       `json:"status"`
	Assignee    *User        `json:"assignee"`
	Priority    *Priority    `json:"priority"`
	Labels      []string     `json:"labels"`
	IssueType   IssueType    `json:"issuetype"`
	Created     string       `json:"created"`
	Updated     string       `json:"updated"`
}

type Status struct {
	Name           string         `json:"name"`
	StatusCategory StatusCategory `json:"statusCategory"`
}

type StatusCategory struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type Priority struct {
	Name string `json:"name"`
}

type IssueType struct {
	Name string `json:"name"`
}

type Board struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type BoardResult struct {
	Values     []Board `json:"values"`
	StartAt    int     `json:"startAt"`
	MaxResults int     `json:"maxResults"`
	Total      int     `json:"total"`
	IsLast     bool    `json:"isLast"`
}

type Worklog struct {
	ID               string       `json:"id"`
	Author           User         `json:"author"`
	Comment          *ADFDocument `json:"comment"`
	Started          string       `json:"started"`
	TimeSpent        string       `json:"timeSpent"`
	TimeSpentSeconds int          `json:"timeSpentSeconds"`
}

type ADFDocument struct {
	Type    string    `json:"type"`
	Version int       `json:"version"`
	Content []ADFNode `json:"content"`
}

type ADFNode struct {
	Type    string    `json:"type"`
	Text    string    `json:"text,omitempty"`
	Content []ADFNode `json:"content,omitempty"`
}

type WorklogResult struct {
	Worklogs   []Worklog `json:"worklogs"`
	Total      int       `json:"total"`
	StartAt    int       `json:"startAt"`
	MaxResults int       `json:"maxResults"`
}

type IssueWorklogSummary struct {
	IssueKey string
	Summary  string
	Total    int
	Entries  []Worklog
}
