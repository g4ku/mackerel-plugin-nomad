package mpnomad

import (
	"flag"

	"github.com/hashicorp/nomad/api"
	mp "github.com/mackerelio/go-mackerel-plugin-helper"
	"github.com/mackerelio/golib/logging"
)

var logger = logging.GetLogger("metrics.plugin.nomad")

// NomadPlugin plugin
type NomadPlugin struct {
	Client *api.Client
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
		"agent.members": {
			Label: "Nomad agent members",
			Unit:  "integer",
			Metrics: []mp.Metrics{
				{Name: "alive", Label: "Alive"},
				{Name: "leaving", Label: "Leaving"},
				{Name: "left", Label: "Left"},
				{Name: "failed", Label: "Failed"},
			},
		},
		"nodes": {
			Label: "Nomad nodes",
			Unit:  "integer",
			Metrics: []mp.Metrics{
				{Name: "initializing", Label: "Initializing"},
				{Name: "ready", Label: "Ready"},
				{Name: "down", Label: "Down"},
				{Name: "ineligible", Label: "Ineligible"},
				{Name: "draining", Label: "Draining"},
			},
		},
	}
}

func (np NomadPlugin) getJobs() ([]*api.JobListStub, error) {
	jobs, _, err := np.Client.Jobs().List(&api.QueryOptions{})
	if err != nil {
		logger.Warningf("Failed to get jobs api", err)
		return nil, err
	}
	return jobs, nil
}

func (np NomadPlugin) getDeployments() ([]*api.Deployment, error) {
	deployments, _, err := np.Client.Deployments().List(&api.QueryOptions{})
	if err != nil {
		logger.Warningf("Failed to get deployments api", err)
		return nil, err
	}
	return deployments, nil
}

func (np NomadPlugin) getAgentMembers() (*api.ServerMembers, error) {
	members, err := np.Client.Agent().Members()
	if err != nil {
		logger.Warningf("Failed to get agent/members api", err)
		return nil, err
	}
	return members, nil
}

func (np NomadPlugin) getNodes() ([]*api.NodeListStub, error) {
	nodes, _, err := np.Client.Nodes().List(&api.QueryOptions{})
	if err != nil {
		logger.Warningf("Failed to get agent/members api", err)
		return nil, err
	}
	return nodes, nil
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
	agentMembers, err := np.getAgentMembers()
	if err != nil {
		return nil, err
	}
	nodes, err := np.getNodes()
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

	// https://github.com/hashicorp/nomad/blob/master/ui/mirage/factories/agent.js#L7
	alive, leaving, left, failed := 0, 0, 0, 0
	for _, member := range agentMembers.Members {
		switch member.Status {
		case "alive":
			alive++
		case "leaving":
			leaving++
		case "left":
			left++
		case "failed":
			failed++
		default:
			logger.Warningf("Unkown agent member status", member.Status)
		}
	}
	result["alive"] = uint64(alive)
	result["leaving"] = uint64(leaving)
	result["left"] = uint64(left)
	result["failed"] = uint64(failed)

	// https://github.com/hashicorp/nomad/blob/master/ui/app/controllers/clients/index.js#L67
	initializing, ready, down, ineligible, draining := 0, 0, 0, 0, 0
	for _, node := range nodes {
		switch node.Status {
		case "initializing":
			initializing++
		case "ready":
			ready++
		case "down":
			down++
		case "ineligible":
			ineligible++
		case "draining":
			draining++
		default:
			logger.Warningf("Unkown node status", node.Status)
		}
	}
	result["initializing"] = uint64(initializing)
	result["ready"] = uint64(ready)
	result["down"] = uint64(down)
	result["ineligible"] = uint64(ineligible)
	result["draining"] = uint64(draining)

	return result, nil
}

// Do the plugin
func Do() {
	optAddress := flag.String("address", "127.0.0.1", "Address")
	optPort := flag.String("port", "4646", "Port")
	flag.Parse()

	cfg := api.DefaultConfig()
	cfg.Address = "http://" + *optAddress + ":" + *optPort

	client, _ := api.NewClient(cfg)

	np := NomadPlugin{
		Client: client,
	}
	helper := mp.NewMackerelPlugin(np)

	helper.Run()
}
