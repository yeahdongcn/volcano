package networktopology

import (
	"github.com/yeahdongcn/topology/pkg/slurm/topology/tree"
	"k8s.io/klog/v2"

	"volcano.sh/volcano/pkg/scheduler/api"
	"volcano.sh/volcano/pkg/scheduler/framework"
)

const (
	// PluginName indicates name of volcano scheduler plugin
	PluginName = "network-topology"

	// TopologyConfigPath is the path to the topology configuration file
	TopologyConfigPath = "topology-config-path"

	// networkExtraScore is the score added to nodes that are selected by the network topology plugin
	networkExtraScore = 1

	PluginWeight = "network-topology.weight"
)

type result struct {
	selectedNodes   []string
	leafSwitchCount uint16
}

type networkTopologyPlugin struct {
	// Arguments given for the plugin
	pluginArguments framework.Arguments

	weight int

	// cacheResults is a map of job UID to the result of the network topology plugin
	cacheResults map[string]result
}

// New return gang plugin
func New(arguments framework.Arguments) framework.Plugin {
	return &networkTopologyPlugin{
		pluginArguments: arguments,
		weight:          calculateWeight(arguments),
		cacheResults:    make(map[string]result),
	}
}

func (ntp *networkTopologyPlugin) Name() string {
	return PluginName
}

func (ntp *networkTopologyPlugin) OnSessionOpen(ssn *framework.Session) {
	klog.V(4).Info("networkTopologyPlugin.OnSessionOpen")

	batchNodeOrderFn := func(task *api.TaskInfo, nodeInfo []*api.NodeInfo) (map[string]float64, error) {
		nodeScores := make(map[string]float64, len(nodeInfo))

		jobInfo := ssn.Jobs[task.Job]
		if result, ok := ntp.cacheResults[string(jobInfo.UID)]; ok {
			for _, node := range result.selectedNodes {
				nodeScores[node] = float64(networkExtraScore * ntp.weight)
			}
			return nodeScores, nil
		}

		miniMember := uint32(jobInfo.PodGroup.Spec.MinMember)
		if miniMember == 0 {
			miniMember = uint32(len(jobInfo.Tasks))
		}

		path, ok := ntp.pluginArguments[TopologyConfigPath].(string)
		if !ok {
			klog.Errorf("Failed to get topology config path from arguments")
			return nodeScores, nil
		}

		err := tree.SwitchRecordValidate(path)
		if err != nil {
			klog.Errorf("Failed to validate switch record from %s: %v", path, err)
			return nodeScores, err
		}

		availableNodes := make([]string, 0, len(nodeInfo))
		for _, node := range nodeInfo {
			availableNodes = append(availableNodes, node.Name)
		}

		klog.V(3).Infof("task: %v/%v, AvailableNodes: %v, miniMember: %v", task.Namespace, task.Name, availableNodes, miniMember)
		if len(availableNodes) < int(miniMember) {
			klog.Errorf("availableNodes count less than miniMember")
			return nodeScores, err
		}

		requiredNodes := make([]string, 0)
		selectedNodes, leafSwitchCount, err := tree.EvalNodesTree(availableNodes, requiredNodes, miniMember)
		if err != nil {
			klog.Errorf("Failed to evaluate nodes tree: %v", err)
			return nodeScores, err
		}

		klog.V(3).Infof("Selected nodes: %v, leaf switch count: %d", selectedNodes, leafSwitchCount)
		ntp.cacheResults[string(jobInfo.UID)] = result{selectedNodes, leafSwitchCount}

		for _, node := range selectedNodes {
			nodeScores[node] = float64(networkExtraScore * ntp.weight)
		}

		return nodeScores, nil
	}
	ssn.AddBatchNodeOrderFn(ntp.Name(), batchNodeOrderFn)
}

func (ntp *networkTopologyPlugin) OnSessionClose(ssn *framework.Session) {
}

func calculateWeight(args framework.Arguments) int {
	/*
	   User Should give networkTopologyWeight in this format(network-topology.weight).

	   actions: "enqueue, reclaim, allocate, backfill, preempt"
	   tiers:
	   - plugins:
	     - name: network-topology
	       arguments:
	         network-topology.weight: 10
	*/
	// Values are initialized to 1.
	weight := 1

	args.GetInt(&weight, PluginWeight)

	return weight
}
