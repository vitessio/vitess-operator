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
    * EtcdCluster ([etcd-operator](https://github.com/coreos/etcd-operator)):
      Vitess needs its own etcd cluster to coordinate its built-in load-balancing
      and automatic shard routing.
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
        * Pod(s) ([vttablet](https://vitess.io/overview/#vttablet)): Within a shard, there may be many Vitess [tablets](https://vitess.io/overview/concepts/#tablet)
          (individual MySQL instances).
          VitessShard acts like an app-specific replacement for StatefulSet,
          creating both Pods and PersistentVolumeClaims.
        * PersistentVolumeClaim(s)
      * **VitessShard** (db1/1)
        * Pod(s) (vttablet)
        * PersistentVolumeClaim(s)
    * **VitessKeyspace** (db2)
      * **VitessShard** (db2/0)
        * Pod(s) (vttablet)
        * PersistentVolumeClaim(s)

## Prerequisites

* Kubernetes 1.8+ is required for its improved CRD support, especially garbage
  collection.
  * This config currently requires a dynamic PersistentVolume provisioner and a
    default StorageClass.
  * The example `my-vitess.yaml` config results in a lot of Pods.
    If the Pods don't schedule due to resource limits, you can try lowering the
    limits, lowering `replicas` values, or removing the `batch` config under
    `tablets`.
* Install [Metacontroller](https://github.com/GoogleCloudPlatform/metacontroller).
* Install [etcd-operator](https://github.com/coreos/etcd-operator) in the
  namespace where you plan to create a VitessCluster.

## Deploy the Operator

You can technically install the Operator into any namespace,
but the references in this example are hard-coded to `vitess` because some
explicit namespace must be specified to ensure the webhooks can be reached
across namespaces.

Note that once the Operator is installed, you can create VitessCluster
objects in any namespace.
The example below loads `my-vitess.yaml` into the default namespace for your
kubectl context.
That's the namespace where etcd-operator also needs to be enabled,
not necessarily the `vitess` namespace.

```sh
kubectl create namespace vitess
kubectl create configmap vitess-operator-hooks -n vitess --from-file=hooks
kubectl apply -f vitess-operator.yaml
```

### Create a VitessCluster

```sh
kubectl apply -f my-vitess.yaml
```

### View the Vitess Dashboard

Wait until the cluster is ready:

```sh
kubectl get vitessclusters -o 'custom-columns=NAME:.metadata.name,READY:.status.conditions[?(@.type=="Ready")].status'
```

You should see:

```console
NAME      READY
vitess    True
```

Start a kubectl proxy:

```sh
kubectl proxy --port=8001
```

Then visit:

```
http://localhost:8001/api/v1/namespaces/default/services/vitess-global-vtctld:web/proxy/app/
```

### Clean Up

```sh
# Delete the VitessCluster object.
kubectl delete -f my-vitess.yaml
# Uninstall the Vitess Operator.
kubectl delete -f vitess-operator.yaml
kubectl delete -n vitess configmap vitess-operator-hooks
# Delete the namespace for the Vitess Operator,
# assuming you created it just for this example.
kubectl delete namespace vitess
```
