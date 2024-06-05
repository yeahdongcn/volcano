# Network Topology-Aware Scheduling

We have some thoughts on network topology-aware scheduling as well, `BUT` limited in the same IDC.

## User Story

Saying we have a 10,000 GPU cluster and each node is connected to a leaf switch through `InfiniBand` or `RoCE`. Each leaf switch connects to spine switches and there might be multiple paths between leaf switches. The network topology is a fat-tree.

End users would like to submit an LLM training job to the cluster. Typically, the job is an MPI job and the communication between nodes is frequent. In most cases, the communication between nodes in the same leaf switch is faster than the communication between nodes in different leaf switches (fewer jumps, less latency).

End users would like to schedule the tasks to the nodes that have optimal network connectivity at the moment. For example, if the tasks are scheduled to the nodes that are connected to the same leaf switch, the communication between the tasks is faster and the job can finish earlier.

We want to schedule the tasks to the nodes that are connected to the same leaf switch, or at least to the nodes that are connected to the same spine switch. This can reduce the network latency and improve the performance.

## Design

We can add a new plugin to the scheduler to support network topology-aware scheduling. The plugin can get the network topology information from the network devices and use the information to schedule the tasks.

### Network Topology Information

The network topology information can be stored in a configmap. The information includes the network topology, the network devices, the connections between the devices, and the latency between the devices.

Similar to how to pick the topology-aware nodes in Slurm, we can use certain tools to discover the network topology alive. For example, we can use `ibnetdiscover` to discover the InfiniBand network topology.

We can have another controller to watch the node changes and spin up a job to discover the network topology on the nodes with InfiniBand. The controller can update the network topology information in the configmap.

### Network Topology-Aware Plugin

The network topology-aware plugin can get the network topology information from the configmap and use the information to schedule the tasks. It works like a score plugin to score the nodes based on the network topology information and pick the best node for a task.

## Initial Code Change

Please see: https://github.com/yeahdongcn/volcano/tree/topo

This is a PoC to show how to add a network topology-aware plugin to the scheduler. See `https://github.com/yeahdongcn/volcano/blob/topo/pkg/scheduler/actions/allocate/allocate_test.go#L160` for the test case.

If anyone is interested in this topic, we can have further discussion.