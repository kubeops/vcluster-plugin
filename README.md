## vCluster Plugin

This [vCluster](https://github.com/loft-sh/vcluster) plugin syncs [CAProviderClass](https://github.com/kubeops/csi-driver-cacerts) crds from the vcluster into the host cluster.

This plugin has been forked from [loft-sh/vcluster-plugin-example](https://github.com/loft-sh/vcluster-plugin-example). For more information how to develop plugins in vcluster and a complete walk through, please refer to the [official vcluster docs](https://www.vcluster.com/docs/v0.19/advanced-topics/plugins-overview).

### Using the Plugin

To use the plugin, create a new vcluster with the `plugin.yaml`:

```
# Install csi-driver-cacerts in host cluster
helm upgrade -i cert-manager-csi-driver-cacerts \
  oci://ghcr.io/appscode-charts/cert-manager-csi-driver-cacerts \
  --version v2024.7.28 \
  -n cert-manager --create-namespace --wait

# Use public plugin.yaml
vcluster create vcluster -n vcluster \
  -f https://github.com/kubeops/vcluster-plugin/raw/master/plugin.yaml
```

This will create a new vcluster with the plugin installed. After that, wait for vcluster to start up and check:

```
# Create a car in the virtual cluster
vcluster connect vcluster -n vcluster -- kubectl apply -f manifests/sample.yaml

# Check if the car was synced to the host cluster
kubectl get caproviderclass -n vcluster
```

### Building the Plugin

To just build the plugin image and push it to the registry, run:

```
# Build
docker build --push -t ghcr.io/appscode/vcluster-plugin:v0.0.1 .

# Multi-arch Build
## Ensure docker builder with multi platform support
docker buildx create \
  --name container \
  --driver=docker-container

## Build & push image
docker build --push \
  --builder container --platform linux/amd64,linux/arm64 \
  -t ghcr.io/appscode/vcluster-plugin:v0.0.1 .
```

Then exchange the image in the `plugin.yaml`.

## Development

General vcluster plugin project structure:
```
.
├── go.mod              # Go module definition
├── go.sum
├── devspace.yaml       # Development environment definition
├── devspace_start.sh   # Development entrypoint script
├── Dockerfile          # Production Dockerfile 
├── main.go             # Go Entrypoint
├── plugin.yaml         # Plugin Helm Values
├── syncers/            # Plugin Syncers
└── manifests/          # Additional plugin resources
```

Before starting to develop, make sure you have installed the following tools on your computer:
- [docker](https://docs.docker.com/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/) with a valid kube context configured
- [helm](https://helm.sh/docs/intro/install/), which is used to deploy vcluster and the plugin
- [vcluster CLI](https://www.vcluster.com/docs/getting-started/setup) v0.6.0 or higher
- [DevSpace](https://devspace.sh/cli/docs/quickstart), which is used to spin up a development environment
- [Go](https://go.dev/dl/) programming language build tools

After successfully setting up the tools, start the development environment with:
```
devspace dev -n vcluster
```

After a while a terminal should show up with additional instructions. Enter the following command to start the plugin:
```
go build -mod vendor -o plugin main.go && /vcluster/syncer start
```

You can now change a file locally in your IDE and then restart the command in the terminal to apply the changes to the plugin.

Delete the development environment with:
```
devspace purge -n vcluster
```
