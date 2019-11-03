module github.com/danielb42/kubeswitch

go 1.13

require (
	github.com/gdamore/tcell v1.3.0
	github.com/rivo/tview v0.0.0-20191018125527-685bf6da76c2
	k8s.io/api v0.0.0-20191016110246-af539daaa43a
	k8s.io/apimachinery v0.0.0-20191004115701-31ade1b30762
	k8s.io/client-go v0.0.0-00010101000000-000000000000
)

replace (
	k8s.io/api => k8s.io/api v0.0.0-20191016110246-af539daaa43a
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191004115701-31ade1b30762
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191016110837-54936ba21026
)
