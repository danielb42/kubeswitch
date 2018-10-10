# kubeswitch
Switch your current kubernetes context and namespace graphically by selecting from a tree. kubeswitch talks to the kubernetes API and does not depend on kubectl. 

![Screenshot](kubeswitch.png)&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;![Demo](demo.gif)

## Install
`go get github.com/danielb42/kubeswitch`

## Config
The location of your `kube.conf` is read from environment variable `KUBECONFIG`.

## Run
Just run `kubeswitch` and select your desired context/namespace.