local k8s = import "k8s.libsonnet";
local vitess = import "vitess.libsonnet";
local metacontroller = import "metacontroller.libsonnet";

{
  local vtgate = self,

  // Filter for vtgate objects among other children of VitessCell.
  matchName(obj)::
    std.endsWith(obj.metadata.name, "-vtgate"),

  matchLabels(obj)::
    k8s.matchLabels(obj, {"vitess.io/component": "vtgate"}),

  // Collections of vtgate objects.
  services(observed, specs)::
    metacontroller.collection(observed, specs, "v1", "Service", vtgate.service)
      + metacontroller.collectionFilter(vtgate.matchName),
  deployments(observed, specs)::
    metacontroller.collection(observed, specs, "apps/v1beta2", "Deployment", vtgate.deployment)
      + metacontroller.collectionFilter(vtgate.matchName),

  // Create/update a Service for vtgate.
  service(observed, spec):: {
    local exposedPorts = [
        {name: "web",  port: 15000},
        {name: "grpc", port: 15999},
    ]+(if "exportMysql" in spec then [{name: "mysql", port:3306}] else []),

    apiVersion: "v1",
    kind: "Service",
    metadata: {
      name: observed.parent.metadata.name + "-vtgate",
      labels: observed.parent.spec.template.metadata.labels,
    },
    spec: {
      selector: observed.parent.spec.selector.matchLabels + {
        "vitess.io/component": "vtgate",
      },
      ports: exposedPorts,
    },
  },

  // Create/update a Deployment for vtgate.
  deployment(observed, spec):: {
    local vtgateFlags = {
      cell: observed.parent.spec.name,
      service_map: "grpc-vtgateservice",
      cells_to_watch: self.cell,
      tablet_types_to_wait: "MASTER,REPLICA",
      gateway_implementation: "discoverygateway",
    } + (if "secret" in spec  then {"mysql_auth_server_static_file": "/credentials/"+spec.secret.key} else {})
      + (if "exportMysql" in spec then {"mysql_server_port": "3306"} else {}),
    local flags = vitess.serverFlags
      + vitess.topoFlags(observed.parent.spec.cluster)
      + vtgateFlags
      + (if "flags" in spec then spec.flags else {}),

    apiVersion: "apps/v1beta2",
    kind: "Deployment",
    metadata: {
      name: observed.parent.metadata.name + "-vtgate",
      labels: observed.parent.spec.template.metadata.labels,
    },
    spec: {
      replicas: spec.replicas,
      selector: observed.parent.spec.selector + {
        matchLabels+: {
          "vitess.io/component": "vtgate",
        },
      },
      template: {
        metadata: {
          labels: observed.parent.spec.template.metadata.labels + {
            "vitess.io/component": "vtgate",
          },
        },
        spec: {
          securityContext: {runAsUser: 999, fsGroup: 999},
          containers: [
            {
              name: "vtgate",
              image: spec.image,
              livenessProbe: {
                httpGet: {path: "/debug/vars", port: 15000},
                initialDelaySeconds: 30,
                timeoutSeconds: 5,
              },
              volumeMounts: [ (if "secret" in spec then {name: "credentials", mountPath: "/credentials"} else {}) ],
              resources: spec.resources,
              command: ["bash", "-c",
                "set -ex; exec /vt/bin/vtgate " +
                vitess.formatFlags(flags)],
            },
          ],
          volumes: (if "secret" in spec then [{name: "credentials", secret: { secretName: spec.secret.name, key: spec.secret.key } }] else []),
        },
      },
    },
  },
}
