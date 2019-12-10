# kubeswitch-lite
[![Go Report Card](https://goreportcard.com/badge/github.com/danielb42/kubeswitch)](https://goreportcard.com/report/github.com/danielb42/kubeswitch) 
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)  

Switch your current kubernetes context and namespace graphically by selecting from a tree. kubeswitch talks to the kubernetes API and does not depend on kubectl.  
The lite-version is for cluster tenants who don't have API-permission to list namespaces and thus cannot use the original kubeswitch program. If you have the permission, you might want to use [kubeswitch](https://github.com/danielb42/kubeswitch/tree/master/cmd/kubeswitch) instead.

![Screenshot](../../kubeswitch.png)&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;![Demo](../../demo.gif)

## Install
```
go install github.com/danielb42/kubeswitch/cmd/kubeswitch-lite
```

## Config
kubeswitch-lite operates on its own kubeconfig file which is a copy of your original config or merged together from multiple other kubeconfigs.  
1. To generate a merged config file, run `kubeswitch-lite --init /path/to/kubeconf1 /path/to/kubeconf2 [...]`.
* `user` objects must be named `user-<namespace>-<cluster>` before merging.
2. `export KUBECONFIG="$HOME/.kube/kubeswitch.yaml"`
3. Your accessible namespaces are read from `~/.kubeswitch_namespaces`, place one namespace name per line there.

## Run
| Run... | to... |
|-|-|
| `kubeswitch-lite` | select context/namespace graphically |  
