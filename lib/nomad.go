package mpnomad

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"

	"github.com/g4ku/mackerel-plugin-nomad/lib/api"
	mp "github.com/mackerelio/go-mackerel-plugin-helper"
	"github.com/mackerelio/golib/logging"
)

var logger = logging.GetLogger("metrics.plugin.nomad")

// NomadPlugin plugin
type NomadPlugin struct {
	Endpoint string
}

// GraphDefinition interface for mackerelplugin
func (np NomadPlugin) GraphDefinition() map[string]mp.Graphs {
	return map[string]mp.Graphs{
		"jobs.#": {
			Label: "Nomad job status",
			Unit:  "integer",
			Metrics: []mp.Metrics{
				{Name: "queued", Label: "Queued"},
				{Name: "complete", Label: "Complete"},
				{Name: "failed", Label: "Failed"},
				{Name: "running", Label: "Running"},
				{Name: "starting", Label: "Starting"},
				{Name: "lost", Label: "Lost"},
			},
		},
		"deployments.#": {
			Label: "Nomad deployments status",
			Unit:  "integer",
			Metrics: []mp.Metrics{
				{Name: "desired_canaries", Label: "DesiredCanaries"},
				{Name: "desired_total", Label: "DesiredTotal"},
				{Name: "placed_allocs", Label: "PlacedAllocs"},
				{Name: "healthy_allocs", Label: "HealthyAllocs"},
				{Name: "unhealthy_allocs", Label: "UnhealthyAllocs"},
			},
		},
	}
}

// fetch REST API
func fetchAPI(endpoint string, api string) ([]byte, error) {
	url := endpoint + api
	resp, err := http.Get(url)
	if err != nil {
		logger.Warningf("HTTP get error:", err)
		return nil, err
	}

	defer resp.Body.Close()
	byteArray, _ := ioutil.ReadAll(resp.Body)
	return byteArray, nil
}

func (np NomadPlugin) getJobs() ([]api.Jobs, error) {
	jsonBytes, err := fetchAPI(np.Endpoint, "/v1/jobs")
	if err != nil {
		return nil, err
	}

	data := new([]api.Jobs)
	if err := json.Unmarshal(jsonBytes, data); err != nil {
		logger.Warningf("JSON Unmarshal error:", err)
		return nil, err
	}
	return *data, nil
}

func (np NomadPlugin) getDeployments() ([]api.Deployments, error) {
	jsonBytes, err := fetchAPI(np.Endpoint, "/v1/deployments")
	if err != nil {
		return nil, err
	}

	data := new([]api.Deployments)
	if err := json.Unmarshal(jsonBytes, data); err != nil {
		logger.Warningf("JSON Unmarshal error:", err)
		return nil, err
	}
	return *data, nil
}

// FetchMetrics interface for mackerelplugin
func (np NomadPlugin) FetchMetrics() (map[string]interface{}, error) {
	jobs, err := np.getJobs()
	if err != nil {
		return nil, err
	}
	deployments, err := np.getDeployments()
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})

	for _, job := range jobs {
		for key, summary := range job.JobSummary.Summary {
			task := job.JobSummary.JobID + "_" + key

			result["jobs."+task+".queued"] = uint64(summary.Queued)
			result["jobs."+task+".complete"] = uint64(summary.Complete)
			result["jobs."+task+".failed"] = uint64(summary.Failed)
			result["jobs."+task+".running"] = uint64(summary.Running)
			result["jobs."+task+".starting"] = uint64(summary.Starting)
			result["jobs."+task+".lost"] = uint64(summary.Lost)
		}
	}

	for _, deployment := range deployments {
		for key, taskGroup := range deployment.TaskGroups {
			task := deployment.JobID + "_" + key

			result["deployments."+task+".desired_canaries"] = uint64(taskGroup.DesiredCanaries)
			result["deployments."+task+".desired_total"] = uint64(taskGroup.DesiredTotal)
			result["deployments."+task+".placed_allocs"] = uint64(taskGroup.PlacedAllocs)
			result["deployments."+task+".healthy_allocs"] = uint64(taskGroup.HealthyAllocs)
			result["deployments."+task+".unhealthy_allocs"] = uint64(taskGroup.UnhealthyAllocs)
		}
	}

	return result, nil
}

// Do the plugin
func Do() {
	optAddress := flag.String("address", "127.0.0.1", "Address")
	optPort := flag.String("port", "4646", "Port")
	flag.Parse()

	endpoint := "http://" + *optAddress + ":" + *optPort

	np := NomadPlugin{
		Endpoint: endpoint,
	}
	helper := mp.NewMackerelPlugin(np)

	helper.Run()
}
