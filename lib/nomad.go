package mpnomad

import (
	"flag"
	"sync"

	"github.com/hashicorp/nomad/api"
	mp "github.com/mackerelio/go-mackerel-plugin"
	"github.com/mackerelio/golib/logging"
)

var logger = logging.GetLogger("metrics.plugin.nomad")

// NomadPlugin plugin
type NomadPlugin struct {
	Client *api.Client
	Tasks  []string
}

// GraphDefinition interface for mackerelplugin
func (np *NomadPlugin) GraphDefinition() map[string]mp.Graphs {
	def := make(map[string]mp.Graphs)
	def["jobs.#"] = mp.Graphs{
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
	}
	def["deployments.#"] = mp.Graphs{
		Label: "Nomad deployments status",
		Unit:  "integer",
		Metrics: []mp.Metrics{
			{Name: "desired_canaries", Label: "DesiredCanaries"},
			{Name: "desired_total", Label: "DesiredTotal"},
			{Name: "placed_allocs", Label: "PlacedAllocs"},
			{Name: "healthy_allocs", Label: "HealthyAllocs"},
			{Name: "unhealthy_allocs", Label: "UnhealthyAllocs"},
		},
	}
	def["agent.members"] = mp.Graphs{
		Label: "Nomad agent members",
		Unit:  "integer",
		Metrics: []mp.Metrics{
			{Name: "alive", Label: "Alive"},
			{Name: "leaving", Label: "Leaving"},
			{Name: "left", Label: "Left"},
			{Name: "failed", Label: "Failed"},
		},
	}
	def["nodes"] = mp.Graphs{
		Label: "Nomad nodes",
		Unit:  "integer",
		Metrics: []mp.Metrics{
			{Name: "initializing", Label: "Initializing"},
			{Name: "ready", Label: "Ready"},
			{Name: "down", Label: "Down"},
			{Name: "ineligible", Label: "Ineligible"},
			{Name: "draining", Label: "Draining"},
		},
	}
	for _, task := range np.Tasks {
		def[task+".#"] = mp.Graphs{
			Label: "Allocation status by task",
			Unit:  "integer",
			Metrics: []mp.Metrics{
				{Name: "cpu_percent", Label: "CPU usage"},
				{Name: "cpu_totalticks", Label: "CPU total ticks"},
				{Name: "memory_rss_bytes", Label: "Memory RSS usage"},
				{Name: "allocated_memory_megabytes", Label: "Memory allocated to tasks"},
			},
		}
	}
	return def
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
		logger.Warningf("Failed to get nodes api", err)
		return nil, err
	}
	return nodes, nil
}

func (np NomadPlugin) getRunningAllocs() ([]*api.AllocationListStub, error) {
	allocs, _, err := np.Client.Allocations().List(&api.QueryOptions{})
	if err != nil {
		logger.Warningf("Failed to get allocations api", err)
		return nil, err
	}

	var result []*api.AllocationListStub
	for _, a := range allocs {
		if a.ClientStatus == "running" {
			result = append(result, a)
		}
	}
	return result, nil
}

func (np NomadPlugin) getAllocInfo(allocID string) (*api.Allocation, error) {
	alloc, _, err := np.Client.Allocations().Info(allocID, &api.QueryOptions{})
	if err != nil {
		logger.Warningf("Failed to get allocation info api", err)
		return nil, err
	}
	return alloc, nil
}

func (np NomadPlugin) getAllocStats(alloc *api.Allocation) (*api.AllocResourceUsage, error) {
	stats, err := np.Client.Allocations().Stats(alloc, &api.QueryOptions{})
	if err != nil {
		logger.Warningf("Failed to get allocation stats api", err)
		return nil, err
	}
	return stats, nil
}

func (np *NomadPlugin) updateTasks(prefixList []string) {
	np.Tasks = removeDuplicate(prefixList)
}

// Sliceの重複要素削除
func removeDuplicate(args []string) []string {
	results := make([]string, 0, len(args))
	encountered := map[string]bool{}
	for i := 0; i < len(args); i++ {
		if !encountered[args[i]] {
			encountered[args[i]] = true
			results = append(results, args[i])
		}
	}
	return results
}

// FetchMetrics interface for mackerelplugin
func (np *NomadPlugin) FetchMetrics() (map[string]float64, error) {
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
	runningAllocs, err := np.getRunningAllocs()
	if err != nil {
		return nil, err
	}

	result := make(map[string]float64)

	for _, job := range jobs {
		for key, summary := range job.JobSummary.Summary {
			task := job.JobSummary.JobID + "_" + key

			result["jobs."+task+".queued"] = float64(summary.Queued)
			result["jobs."+task+".complete"] = float64(summary.Complete)
			result["jobs."+task+".failed"] = float64(summary.Failed)
			result["jobs."+task+".running"] = float64(summary.Running)
			result["jobs."+task+".starting"] = float64(summary.Starting)
			result["jobs."+task+".lost"] = float64(summary.Lost)
		}
	}

	for _, deployment := range deployments {
		for key, taskGroup := range deployment.TaskGroups {
			task := deployment.JobID + "_" + key

			result["deployments."+task+".desired_canaries"] = float64(taskGroup.DesiredCanaries)
			result["deployments."+task+".desired_total"] = float64(taskGroup.DesiredTotal)
			result["deployments."+task+".placed_allocs"] = float64(taskGroup.PlacedAllocs)
			result["deployments."+task+".healthy_allocs"] = float64(taskGroup.HealthyAllocs)
			result["deployments."+task+".unhealthy_allocs"] = float64(taskGroup.UnhealthyAllocs)
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
	result["alive"] = float64(alive)
	result["leaving"] = float64(leaving)
	result["left"] = float64(left)
	result["failed"] = float64(failed)

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
	result["initializing"] = float64(initializing)
	result["ready"] = float64(ready)
	result["down"] = float64(down)
	result["ineligible"] = float64(ineligible)
	result["draining"] = float64(draining)

	var w sync.WaitGroup
	var mu sync.RWMutex
	prefixList := make([]string, 0)
	for _, a := range runningAllocs {
		w.Add(1)
		go func(a *api.AllocationListStub) {
			defer w.Done()
			alloc, err := np.getAllocInfo(a.ID)
			if err != nil {
				return
			}
			stats, err := np.getAllocStats(alloc)
			if err != nil {
				return
			}
			for key := range a.TaskStates {
				prefix := a.JobID + "_" + a.TaskGroup + "_" + key
				uniqueKey := prefix + "." + a.ID[:8]
				mu.Lock()
				prefixList = append(prefixList, prefix)
				result[uniqueKey+".cpu_percent"] = stats.Tasks[key].ResourceUsage.CpuStats.Percent
				result[uniqueKey+".cpu_totalticks"] = stats.Tasks[key].ResourceUsage.CpuStats.TotalTicks
				result[uniqueKey+".memory_rss_bytes"] = float64(stats.Tasks[key].ResourceUsage.MemoryStats.RSS)
				result[uniqueKey+".allocated_memory_megabytes"] = float64(*alloc.TaskResources[key].MemoryMB)
				mu.Unlock()
			}
		}(a)
	}
	w.Wait()
	np.updateTasks(prefixList)

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

	mp.NewMackerelPlugin(&np).Run()
}
