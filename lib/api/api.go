package api

type Jobs struct {
	ID                string
	ParentID          string
	Name              string
	Type              string
	Priority          int
	Status            string
	StatusDescription string
	JobSummary        struct {
		JobID   string
		Summary map[string]struct {
			Queued   int
			Complete int
			Failed   int
			Running  int
			Starting int
			Lost     int
		}
		Children struct {
			Pending int
			Running int
			Dead    int
		}
		CreateIndex int
		ModifyIndex int
	}
	CreateIndex    int
	ModifyIndex    int
	JobModifyIndex int
}

type Deployments struct {
	ID                 string
	JobID              string
	JobVersion         int
	JobModifyIndex     int
	JobSpecModifyIndex int
	JobCreateIndex     int
	TaskGroups         map[string]struct {
		Promoted        bool
		DesiredCanaries int
		DesiredTotal    int
		PlacedAllocs    int
		HealthyAllocs   int
		UnhealthyAllocs int
	}
	Status            string
	StatusDescription string
	CreateIndex       int
	ModifyIndex       int
}
