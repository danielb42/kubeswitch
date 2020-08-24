# kubeswitch

![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/danielb42/kubeswitch)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/danielb42/kubeswitch)
[![Go Report Card](https://goreportcard.com/badge/github.com/danielb42/kubeswitch)](https://goreportcard.com/report/github.com/danielb42/kubeswitch)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)  

Switch your current kubernetes context and namespace graphically by selecting from a tree.  
kubeswitch talks to the kubernetes API and does not depend on kubectl.

![Demo](https://raw.githubusercontent.com/danielb42/kubeswitch/master/demo.gif)

## Install

```golang
GO111MODULE=on go get github.com/danielb42/kubeswitch/cmd/kubeswitch
```

**Note for Non-Admin Users:** If you are a cluster tenant without API-permission to list namespaces, kubeswitch won't work for you (as it can't retrieve available namespaces). Sorry, there's not much we can do about that.

## Config

Read from the default location `~/.kube/config`. If not present, the location is read from environment variable `KUBECONFIG` (remember to `export`). That env variable can contain multiple locations separated by `:` from where configs are merged together.

## Run

| Run... | to... |
|-|-|
| `kubeswitch` | select context/namespace graphically |  
| `kubeswitch <namespace>` | switch to namespace in current context quickly |  
| `kubeswitch <context> <namespace>`<br />`kubeswitch <context>/<namespace>` | switch to context/namespace |
