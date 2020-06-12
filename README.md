# Deprecation notice

This repository is deprecated, and will soon be archived.
Instructions for the new operator can be found [here](https://vitess.io/docs/get-started/operator/).

# Vitess Operator

The Vitess Operator provides automation that simplifies the administration
of [Vitess](https://vitess.io) clusters on Kubernetes.

The Operator installs a custom resource for objects of the custom type
VitessCluster.
This custom resource allows you to configure the high-level aspects of
your Vitess deployment, while the details of how to run Vitess on Kubernetes
are abstracted and automated.

## Vitess Components

A typical VitessCluster object might expand to the following tree once it's
fully deployed.
Objects in **bold** are custom resource kinds defined by this Operator.

* **VitessCluster**: The top-level specification for a Vitess cluster.
  This is the only one the user creates.
  * **VitessCell**: Each Vitess [cell](https://vitess.io/overview/concepts/#cell-data-center)
    represents an independent failure domain (e.g. a Zone or Availability Zone).
    * Lockserver ([etcd-operator](https://github.com/coreos/etcd-operator)):
      Vitess needs its own etcd cluster to coordinate its built-in load-balancing
      and automatic shard routing. Vitess supports multiple lockservers but the operator
      only supports etcd right now.
    * Deployment ([orchestrator](https://github.com/github/orchestrator)):
      An optional automated failover tool that works with Vitess.
    * Deployment ([vtctld](https://vitess.io/overview/#vtctld)):
      A pool of stateless Vitess admin servers, which serve a dashboard UI as well
      as being an endpoint for the Vitess CLI tool (vtctlclient).
    * Deployment ([vtgate](https://vitess.io/overview/#vtgate)):
      A pool of stateless Vitess query routers.
      The client application can use any one of these vtgate Pods as the entry
      point into Vitess, through a MySQL-compatible interface.
    * **VitessKeyspace** (db1): Each Vitess [keyspace](https://vitess.io/overview/concepts/#keyspace)
      is a logical database that may be composed of many MySQL databases (shards).
      * **VitessShard** (db1/0): Each Vitess [shard](https://vitess.io/overview/concepts/#shard)
      is a single-master tree of replicating MySQL instances.
        * StatefulSet(s) ([vttablet](https://vitess.io/overview/#vttablet)): Within a shard, there may be many Vitess [tablets](https://vitess.io/overview/concepts/#tablet)
          (individual MySQL instances).
        * PersistentVolumeClaim(s)
      * **VitessShard** (db1/1)
        * StatefulSet(s) (vttablet)
        * PersistentVolumeClaim(s)
    * **VitessKeyspace** (db2)
      * **VitessShard** (db2/0)
        * StatefulSet(s) (vttablet)
        * PersistentVolumeClaim(s)

## Prerequisites

* Kubernetes 1.8+ is required for its improved CRD support, especially garbage
  collection.
  * This config currently requires a dynamic PersistentVolume provisioner and a
    default StorageClass.
* [etcd-operator](https://github.com/coreos/etcd-operator)

## Deploy the Operator

Once the Operator is installed, you can create VitessCluster
objects in any namespace as long as the etcd operator is runing
in that namespace or is running [clusterwide](https://github.com/coreos/etcd-operator/blob/master/doc/user/clusterwide.md) mode.

```sh
kubectl apply -R -f deploy
```

### Create a VitessCluster

```sh
kubectl apply -f my-vitess.yaml
```

### View the Vitess Dashboards

Wait until the cluster is ready:

```sh
kubectl get vitessclusters -o 'custom-columns=NAME:.metadata.name,READY:.status.phase'
```

You should see:

```console
NAME      PHASE
vitess    Ready
```

Start a kubectl proxy:

```sh
kubectl proxy --port=8001
```

Then visit:

```
http://localhost:8001/api/v1/namespaces/default/services/vt-zone1-vtctld:web/proxy/app/
```

### Clean Up

```sh
# Delete the VitessCluster  and etcd objects
kubectl delete -f my-vitess.yaml
# Uninstall the Vitess Operator
kubectl delete -R -f deploy
```

## TODO

- [x] Create a StatefulSet for each VitessTablet in a VitessCluster
- [x] Create a Job to elect the initial master in each VitessShard
- [X] Fix parenting and normalization
- [x] Create vtctld Deployment and Service
- [X] Create vttablet service
- [X] Create vtgate Deployment and Service
- [ ] Create PodDisruptionBudgets
- [ ] Reconcile all the things!
- [ ] Label pods when they become shard masters
- [ ] Add the ability to automatically merge/split a shard
- [ ] Add the ability to automatically export/import resources from embedded objects to separate objects and back
- [ ] Move shard master election into the operator

## Dev

- Install the [operator sdk](https://github.com/operator-framework/operator-sdk)
- Configure local kubectl access to a test Kubernetes cluster
- Create the CRDs in your Kubernetes cluster
    - `kubectl apply -f deploy/crds`
- Run the operator locally
    - `operator-sdk up local`
- Create the sample cluster
    - `kubectl create -f my-vitess.yaml`
